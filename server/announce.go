/*
 * This file is part of Chihaya.
 *
 * Chihaya is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Chihaya is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.
 */

package server

import (
	"bytes"
	"chihaya/config"
	cdb "chihaya/database"
	"chihaya/util"
	"encoding/binary"
	"fmt"
	"github.com/zeebo/bencode"
	"log"
	"math"
	"net"
	"strconv"
	"time"
)

func whitelisted(peerId string, db *cdb.Database) uint32 {
	db.WhitelistMutex.RLock()
	defer db.WhitelistMutex.RUnlock()

	var widLen int
	var i int
	var matched bool

	for id, whitelistedId := range db.Whitelist {
		widLen = len(whitelistedId)
		if widLen <= len(peerId) {
			matched = true
			for i = 0; i < widLen; i++ {
				if peerId[i] != whitelistedId[i] {
					matched = false
					break
				}
			}
			if matched {
				return id
			}
		}
	}
	return 0
}

func hasHitAndRun(db *cdb.Database, userId uint64, torrentId uint64) bool {
	hnr := cdb.UserTorrentPair{
		UserId:    userId,
		TorrentId: torrentId,
	}
	_, exists := db.HitAndRuns[hnr]
	return exists
}

func announce(params *queryParams, user *cdb.User, ipAddr string, db *cdb.Database, buf *bytes.Buffer) {
	var exists bool

	// Mandatory parameters
	infoHash, _ := params.get("info_hash")
	peerId, _ := params.get("peer_id")
	port, portExists := params.getUint64("port")
	uploaded, uploadedExists := params.getUint64("uploaded")
	downloaded, downloadedExists := params.getUint64("downloaded")
	left, leftExists := params.getUint64("left")

	if !(infoHash != "" && peerId != "" && portExists && uploadedExists && downloadedExists && leftExists) {
		failure("Malformed request - missing mandatory param", buf, 1*time.Hour)
		return
	}

	client_id := whitelisted(peerId, db)
	if 0 == client_id {
		failure("Your client is not approved", buf, 1*time.Hour)
		return
	}

	// TODO: better synchronization strategy for announces (like per user mutexes)
	db.TorrentsMutex.Lock()
	defer db.TorrentsMutex.Unlock()

	torrent, exists := db.Torrents[infoHash]
	if !exists {
		failure("This torrent does not exist", buf, 30*time.Second)
		return
	}

	if torrent.Status == 1 && left == 0 {
		log.Printf("Unpruning torrent %d", torrent.Id)
		db.UnPrune(torrent)
		torrent.Status = 0
	} else if torrent.Status != 0 {
		failure(fmt.Sprintf("This torrent does not exist (status: %d, left: %d)", torrent.Status, left), buf, 5*time.Minute)
		return
	}

	now := time.Now().Unix()

	// Optional parameters
	event, _ := params.get("event")

	var numWantStr string
	var numWant int
	numWantStr, exists = params.get("numwant")
	if !exists {
		numWant = 25
	} else {
		numWant64, _ := strconv.ParseInt(numWantStr, 10, 32)
		numWant = int(numWant64)
		if numWant > 50 {
			numWant = 50
		} else if numWant < 0 {
			numWant = 25
		}
	}

	// Match or create peer
	var peer *cdb.Peer
	newPeer := false
	seeding := false
	active := true
	completed := event == "completed"
	peerKey := fmt.Sprintf("%d-%s", user.Id, peerId)

	if left > 0 {
		if user.DisableDownload {
			// only disable download if the torrent doesn't have a HnR against it
			if !hasHitAndRun(db, user.Id, torrent.Id) {
				failure("Your download privileges are disabled.", buf, 1*time.Hour)
				return
			}
		}
		peer, exists = torrent.Leechers[peerKey]
		if !exists {
			newPeer = true
			peer = &cdb.Peer{}
			torrent.Leechers[peerKey] = peer
		}
	} else if completed {
		peer, exists = torrent.Leechers[peerKey]
		if !exists {
			newPeer = true
			peer = &cdb.Peer{}
			torrent.Seeders[peerKey] = peer
		} else {
			// They're a seeder now
			torrent.Seeders[peerKey] = peer
			delete(torrent.Leechers, peerKey)
		}
		seeding = true
	} else { // Previously completed (probably)
		peer, exists = torrent.Seeders[peerKey]
		if !exists {
			peer, exists = torrent.Leechers[peerKey]
			if !exists {
				newPeer = true
				peer = &cdb.Peer{}
				torrent.Seeders[peerKey] = peer
			} else {
				// They're a seeder now.. Broken client? Unreported snatch?
				torrent.Seeders[peerKey] = peer
				delete(torrent.Leechers, peerKey)
				// completed = true // TODO: not sure if this will result in over-reported snatches
			}
		}
		seeding = true
	}

	// Update peer info/stats
	if newPeer {
		peer.Id = peerId
		peer.UserId = user.Id
		peer.TorrentId = torrent.Id
		peer.StartTime = now
		peer.LastAnnounce = now
		peer.Uploaded = uploaded
		peer.Downloaded = downloaded
	}

	rawDeltaUpload := int64(uploaded) - int64(peer.Uploaded)
	rawDeltaDownload := int64(downloaded) - int64(peer.Downloaded)

	// If a user restarts a torrent, their delta may be negative, attenuating this to 0 should be fine for stats purposes
	if rawDeltaUpload < 0 {
		rawDeltaUpload = 0
	}
	if rawDeltaDownload < 0 {
		rawDeltaDownload = 0
	}

	var deltaDownload int64
	if !config.GlobalFreeleech {
		deltaDownload = int64(float64(rawDeltaDownload) * math.Abs(user.DownMultiplier) * math.Abs(torrent.DownMultiplier))
	}
	deltaUpload := int64(float64(rawDeltaUpload) * math.Abs(user.UpMultiplier) * math.Abs(torrent.UpMultiplier))

	peer.Uploaded = uploaded
	peer.Downloaded = downloaded
	peer.Left = left
	peer.Seeding = seeding

	deltaTime := now - peer.LastAnnounce
	if deltaTime > 2*int64(config.AnnounceInterval.Seconds()) {
		deltaTime = 0
	}

	var deltaSeedTime int64
	if seeding {
		deltaSeedTime = now - peer.LastAnnounce
	}
	if deltaSeedTime > 2*int64(config.AnnounceInterval.Seconds()) {
		deltaSeedTime = 0
	}

	peer.LastAnnounce = now
	// update torrent last_action only if announced action is seeding - allows dead torrents without seeder but with leecher
	// to be proeprly pruned
	if seeding {
		torrent.LastAction = now
	}

	// Handle events
	var deltaSnatch uint64
	if event == "stopped" {
		/*  We can remove the peer from the list and still have their stats be recorded,
		since we still have a reference to their object. After flushing, all references
		should be gone, allowing the peer to be GC'd.  */
		if seeding {
			delete(torrent.Seeders, peerKey)
		} else {
			delete(torrent.Leechers, peerKey)
		}

		active = false
	} else if completed {
		db.RecordSnatch(peer, now)
		deltaSnatch = 1
	}

	/*
	 * Generate compact ip/port
	 * Future TODO: possible IPv6 support
	 */
	ipBytes := net.ParseIP(ipAddr)
	if nil == ipBytes {
		failure("Malformed IP address", buf, 1*time.Hour)
		return
	}

	ipBytes = ipBytes.To4()
	if nil == ipBytes {
		failure("IPv4 address required (sorry!)", buf, 1*time.Hour)
		return
	}

	// convers in a way equivalent to PHP's ip2long
	ipLong := binary.BigEndian.Uint32(ipBytes)

	peer.Addr = []byte{ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3], byte(port >> 8), byte(port & 0xff)}
	peer.Port = uint(port)
	peer.IpAddr = ipAddr
	if user.TrackerHide {
		peer.Ip = 0
	} else {
		peer.Ip = ipLong
	}
	peer.ClientId = client_id

	// If the channels are already full, record* blocks until a flush occurs
	db.RecordTorrent(torrent, deltaSnatch)
	db.RecordTransferHistory(peer, rawDeltaUpload, rawDeltaDownload, deltaTime, deltaSeedTime, deltaSnatch, active)
	db.RecordUser(user, rawDeltaUpload, rawDeltaDownload, deltaUpload, deltaDownload)
	record(peer.TorrentId, user.Id, rawDeltaUpload, rawDeltaDownload, uploaded, event, ipAddr)
	db.RecordTransferIp(peer, rawDeltaUpload, rawDeltaDownload)

	// Generate response
	seedCount := len(torrent.Seeders)
	leechCount := len(torrent.Leechers)
	snatchCount := torrent.Snatched

	respData := make(map[string]interface{})
	respData["complete"] = seedCount
	respData["incomplete"] = leechCount
	respData["downloaded"] = snatchCount
	respData["min interval"] = config.MinAnnounceInterval / time.Second                                                  // Assuming seconds
	respData["interval"] = (config.AnnounceInterval + time.Duration(util.Min(600, seedCount))*time.Second) / time.Second // Assuming seconds

	if numWant > 0 && active {

		compactString, exists := params.get("compact")
		compact := !exists || compactString != "0" // Default to being compact

		noPeerIdString, exists := params.get("no_peer_id")
		noPeerId := exists && noPeerIdString == "1"

		var peerCount int
		if seeding {
			peerCount = util.Min(numWant, leechCount)
		} else {
			peerCount = util.Min(numWant, leechCount+seedCount-1)
		}

		peersToSend := make([]*cdb.Peer, 0, peerCount)

		/*
		* The iteration is already "random", so we don't need to randomize ourselves:
		* Each time an element is inserted into the map, it gets a some arbitrary position for iteration
		* Each time you range over the map, it starts at a random offset into the map's elements
		*/

		if seeding {
			for _, leech := range torrent.Leechers {
				if len(peersToSend) >= numWant {
					break
				}
				if leech.UserId == peer.UserId {
					continue
				}
				peersToSend = append(peersToSend, leech)
			}
		} else {
			for _, seed := range torrent.Seeders {
				if len(peersToSend) >= numWant {
					break
				}
				if seed.UserId == peer.UserId {
					continue
				}
				peersToSend = append(peersToSend, seed)
			}

			for _, leech := range torrent.Leechers {
				if len(peersToSend) >= numWant {
					break
				}
				if leech.UserId == peer.UserId {
					continue
				}
				peersToSend = append(peersToSend, leech)
			}
		}

		if compact {
			var peerBuff bytes.Buffer
			for _, other := range peersToSend {
				peerBuff.Write(other.Addr)
			}
			respData["peers"] = peerBuff.String()
		} else {
			peerList := make([]map[string]interface{}, len(peersToSend))
			for i, other := range peersToSend {
				peerMap := make(map[string]interface{})
				peerMap["ip"] = other.IpAddr
				peerMap["port"] = other.Port
				if !noPeerId {
					peerMap["peer id"] = other.Id
				}
				peerList[i] = peerMap
			}
			respData["peers"] = peerList
		}
	}
	bufdata, err := bencode.EncodeBytes(respData)
	if err != nil {
		panic(err)
	}
	buf.Write(bufdata)
}

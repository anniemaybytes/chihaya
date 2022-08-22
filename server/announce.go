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
	"chihaya/database"
	cdb "chihaya/database/types"
	"chihaya/log"
	"chihaya/record"
	"chihaya/server/params"
	"chihaya/util"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/zeebo/bencode"
)

var (
	announceInterval       int
	minAnnounceInterval    int
	peerInactivityInterval int
	maxAccounceDrift       int
	defaultNumWant         int
	maxNumWant             int

	strictPort bool

	privateIPBlocks []*net.IPNet
)

func init() {
	intervalsConfig := config.Section("intervals")
	announceConfig := config.Section("announce")

	announceInterval, _ = intervalsConfig.GetInt("announce", 1800)
	minAnnounceInterval, _ = intervalsConfig.GetInt("min_announce", 900)
	peerInactivityInterval, _ = intervalsConfig.GetInt("peer_inactivity", 3900)
	maxAccounceDrift, _ = intervalsConfig.GetInt("announce_drift", 300)

	strictPort, _ = announceConfig.GetBool("strict_port", false)
	defaultNumWant, _ = announceConfig.GetInt("numwant", 25)
	maxNumWant, _ = announceConfig.GetInt("max_numwant", 50)

	for _, cidr := range []string{
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"100.64.0.0/10",  // RFC6598
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Error.Printf("IP parse error on %q: %v", cidr, err)
			log.WriteStack()
		} else {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}
}

func getPublicIPV4(ipAddr string, exists bool) (string, bool) {
	if !exists { // Already does not exist, fail
		return ipAddr, exists
	}

	ip := net.ParseIP(ipAddr)
	if ip == nil { // Invalid IP provided, fail
		return ipAddr, false
	}

	if ip.To4() == nil { // IPv6 provided, fail
		return ipAddr, false
	}

	private := false

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		private = true
	} else {
		for _, block := range privateIPBlocks {
			if block.Contains(ip) {
				private = true
				break
			}
		}
	}

	return ipAddr, !private
}

func clientApproved(peerID string, db *database.Database) (uint16, bool) {
	db.ClientsMutex.RLock()
	defer db.ClientsMutex.RUnlock()

	var (
		widLen, i int
		matched   bool
	)

	for id, clientID := range db.Clients {
		widLen = len(clientID)
		if widLen <= len(peerID) {
			matched = true

			for i = 0; i < widLen; i++ {
				if peerID[i] != clientID[i] {
					matched = false
					break
				}
			}

			if matched {
				return id, true
			}
		}
	}

	return 0, false
}

func announce(qs string, header http.Header, remoteAddr string, user *cdb.User,
	db *database.Database, buf io.Writer) {
	qp, err := params.ParseQuery(qs)
	if err != nil {
		panic(err)
	}

	// Mandatory parameters
	infoHashes := qp.InfoHashes()
	peerID, _ := qp.Get("peer_id")
	port, portExists := qp.GetUint16("port")
	uploaded, uploadedExists := qp.GetUint64("uploaded")
	downloaded, downloadedExists := qp.GetUint64("downloaded")
	left, leftExists := qp.GetUint64("left")

	if infoHashes == nil {
		failure("Malformed request - missing info_hash", buf, 1*time.Hour)
		return
	} else if len(infoHashes) > 1 {
		failure("Malformed request - multiple info_hash values provided", buf, 1*time.Hour)
		return
	}

	if peerID == "" {
		failure("Malformed request - missing peer_id", buf, 1*time.Hour)
		return
	}

	if len(peerID) != 20 {
		failure("Malformed request - invalid peer_id", buf, 1*time.Hour)
		return
	}

	if !portExists {
		failure("Malformed request - missing port", buf, 1*time.Hour)
		return
	}

	if strictPort && port < 1024 || port > 65535 {
		failure(fmt.Sprintf("Malformed request - port outside of acceptable range (port: %d)", port), buf, 1*time.Hour)
		return
	}

	if !uploadedExists {
		failure("Malformed request - missing uploaded", buf, 1*time.Hour)
		return
	}

	if !downloadedExists {
		failure("Malformed request - missing downloaded", buf, 1*time.Hour)
		return
	}

	if !leftExists {
		failure("Malformed request - missing left", buf, 1*time.Hour)
		return
	}

	ipAddr := func() string {
		ipV4, existsV4 := getPublicIPV4(qp.Get("ipv4")) // First try to get IPv4 address if client sent it
		ip, exists := getPublicIPV4(qp.Get("ip"))       // ... then try to get public IP if sent by client

		// Fail if ip and ipv4 are not same, and both are provided
		if existsV4 && exists && ip != ipV4 {
			return ""
		}

		if existsV4 {
			return ipV4
		}

		if exists {
			return ip
		}

		// Check for proxied IP header
		proxyHeader, exists := config.Section("http").Get("proxy_header", "")
		if exists {
			ips, exists := header[proxyHeader]
			if exists && len(ips) > 0 {
				return ips[0]
			}
		}

		// Check for IP in socket
		portIndex := strings.LastIndex(remoteAddr, ":")
		if portIndex != -1 {
			return remoteAddr[0:portIndex]
		}

		// Everything else failed
		return ""
	}()

	ipBytes := net.ParseIP(ipAddr).To4()
	if nil == ipBytes {
		failure(fmt.Sprintf("Failed to parse IP address (ip: %s)", ipAddr), buf, 1*time.Hour)
		return
	}

	clientID, matched := clientApproved(peerID, db)
	if !matched {
		failure(fmt.Sprintf("Your client is not approved (peer_id: %s)", peerID), buf, 1*time.Hour)
		return
	}

	db.TorrentsMutex.Lock()
	defer db.TorrentsMutex.Unlock()

	torrent, exists := db.Torrents[infoHashes[0]]
	if !exists {
		failure("This torrent does not exist", buf, 30*time.Second)
		return
	}

	if torrent.Status == 1 && left == 0 {
		log.Info.Printf("Unpruning torrent %d", torrent.ID)
		db.UnPrune(torrent)
		torrent.Status = 0
	} else if torrent.Status != 0 {
		failure(fmt.Sprintf("This torrent does not exist (status: %d, left: %d)", torrent.Status, left), buf, 5*time.Minute)
		return
	}

	numWant, exists := qp.GetUint16("numwant")
	if !exists {
		numWant = uint16(defaultNumWant)
	} else if numWant > uint16(maxNumWant) {
		numWant = uint16(maxNumWant)
	}

	var (
		peer    *cdb.Peer
		now     = time.Now().Unix()
		newPeer = false
		seeding = false
		active  = true
	)

	event, _ := qp.Get("event")
	completed := event == "completed"
	peerKey := fmt.Sprintf("%d-%s", user.ID, peerID)

	if left > 0 {
		if isDisabledDownload(db, user, torrent) {
			failure("Your download privileges are disabled", buf, 1*time.Hour)
			return
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
			// Previously tracked peer is now a seeder
			torrent.Seeders[peerKey] = peer
			delete(torrent.Leechers, peerKey)
		}
		seeding = true
	} else {
		peer, exists = torrent.Seeders[peerKey]
		if !exists {
			peer, exists = torrent.Leechers[peerKey]
			if !exists {
				newPeer = true
				peer = &cdb.Peer{}
				torrent.Seeders[peerKey] = peer
			} else {
				/* Previously tracked peer is now a seeder, however we never received their "completed" event.
				Broken client? Unreported snatch? Cross-seeding? Let's not report it as snatch to avoid
				over-reporting for cross-seeding */
				torrent.Seeders[peerKey] = peer
				delete(torrent.Leechers, peerKey)
				// Let's not report it as snatch to avoid over-reporting for cross-seeding
			}
		}
		seeding = true
	}

	// Update peer info/stats
	if newPeer {
		peer.ID = peerID
		peer.UserID = user.ID
		peer.TorrentID = torrent.ID
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

	torrentGroupDownMultiplier := 1.0
	torrentGroupUpMultiplier := 1.0

	if torrentGroupFreeleech, exists := db.TorrentGroupFreeleech[torrent.Group]; exists {
		torrentGroupDownMultiplier = torrentGroupFreeleech.DownMultiplier
		torrentGroupUpMultiplier = torrentGroupFreeleech.UpMultiplier
	}

	var deltaDownload int64
	if !database.GlobalFreeleech {
		deltaDownload = int64(float64(rawDeltaDownload) * math.Abs(user.DownMultiplier) *
			math.Abs(torrentGroupDownMultiplier) * math.Abs(torrent.DownMultiplier))
	}

	deltaUpload := int64(float64(rawDeltaUpload) * math.Abs(user.UpMultiplier) *
		math.Abs(torrentGroupUpMultiplier) * math.Abs(torrent.UpMultiplier))
	peer.Uploaded = uploaded
	peer.Downloaded = downloaded
	peer.Left = left
	peer.Seeding = seeding
	deltaTime := now - peer.LastAnnounce

	if deltaTime > int64(peerInactivityInterval) {
		deltaTime = 0
	}

	var deltaSeedTime int64
	if seeding {
		deltaSeedTime = now - peer.LastAnnounce
	}

	if deltaSeedTime > int64(peerInactivityInterval) {
		deltaSeedTime = 0
	}

	peer.LastAnnounce = now
	/* Update torrent last_action only if announced action is seeding.
	This allows dead torrents without seeder but with leecher to be proeprly pruned */
	if seeding {
		torrent.LastAction = now
	}

	var deltaSnatch uint8

	if event == "stopped" {
		/* We can remove the peer from the list and still have their stats be recorded,
		since we still have a reference to their object. After flushing, all references
		should be gone, allowing the peer to be GC'd. */
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

	peer.Addr = []byte{ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3], byte(port >> 8), byte(port & 0xff)}
	peer.Port = port
	peer.IPAddr = ipAddr

	if user.TrackerHide {
		peer.IP = 2130706433 // 127.0.0.1
	} else {
		peer.IP = binary.BigEndian.Uint32(ipBytes)
	}

	peer.ClientID = clientID

	// If the channels are already full, record* blocks until a flush occurs
	db.RecordTorrent(torrent, deltaSnatch)
	db.RecordTransferHistory(peer, rawDeltaUpload, rawDeltaDownload, deltaTime, deltaSeedTime, deltaSnatch, active)
	db.RecordUser(user, rawDeltaUpload, rawDeltaDownload, deltaUpload, deltaDownload)
	record.Record(
		peer.TorrentID,
		user.ID,
		ipAddr,
		port,
		event,
		seeding,
		rawDeltaUpload,
		rawDeltaDownload,
		uploaded,
		downloaded,
		left)
	db.RecordTransferIP(peer, rawDeltaUpload, rawDeltaDownload)

	// Generate response
	seedCount := len(torrent.Seeders)
	leechCount := len(torrent.Leechers)
	snatchCount := torrent.Snatched

	respData := make(map[string]interface{})
	respData["complete"] = seedCount
	respData["incomplete"] = leechCount
	respData["downloaded"] = snatchCount
	respData["min interval"] = minAnnounceInterval

	/* We ask clients to announce each interval seconds. In order to spread the load on tracker,
	we will vary the interval given to client by random number of seconds between 0 and value
	specified in config */
	announceDrift := util.Rand(0, maxAccounceDrift)
	respData["interval"] = announceInterval + announceDrift

	if numWant > 0 && active {
		compactString, exists := qp.Get("compact")
		compact := !exists || compactString != "0" // Defaults to being compact

		noPeerIDString, exists := qp.Get("no_peer_id")
		noPeerID := exists && noPeerIDString == "1"

		var peerCount int
		if seeding {
			peerCount = util.Min(int(numWant), leechCount)
		} else {
			peerCount = util.Min(int(numWant), leechCount+seedCount-1)
		}

		peersToSend := make([]*cdb.Peer, 0, peerCount)

		/*
		 * The iteration is already "random", so we don't need to randomize ourselves:
		 * - Each time an element is inserted into the map, it gets a some arbitrary position for iteration
		 * - Each time you range over the map, it starts at a random offset into the map's elements
		 */
		if seeding {
			for _, leech := range torrent.Leechers {
				if len(peersToSend) >= int(numWant) {
					break
				}

				if leech.UserID == peer.UserID {
					continue
				}

				peersToSend = append(peersToSend, leech)
			}
		} else {
			/* Send only one peer per user. This is to ensure that users seeding at multiple locations don't end up
			exclusively acting as peers. */
			uniqueSeeders := make(map[uint32]*cdb.Peer)
			for _, seed := range torrent.Seeders {
				if len(peersToSend) >= int(numWant) {
					break
				}
				if seed.UserID == peer.UserID {
					continue
				}
				if _, exists = uniqueSeeders[seed.UserID]; !exists {
					uniqueSeeders[seed.UserID] = seed
					peersToSend = append(peersToSend, seed)
				}
			}
			for _, leech := range torrent.Leechers {
				if len(peersToSend) >= int(numWant) {
					break
				}
				if leech.UserID == peer.UserID {
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
				peerMap["ip"] = other.IPAddr
				peerMap["port"] = other.Port
				if !noPeerID {
					peerMap["peer id"] = other.ID
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

	if _, err = buf.Write(bufdata); err != nil {
		panic(err)
	}
}

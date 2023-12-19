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
	"fmt"
	"log/slog"
	"math"
	"net"
	"strings"
	"time"

	"chihaya/config"
	"chihaya/database"
	cdb "chihaya/database/types"
	"chihaya/record"
	"chihaya/server/params"
	"chihaya/util"

	"github.com/valyala/fasthttp"
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
			slog.Error("failed to parse cidr", "cidr", cidr, "err", err)
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

func announce(ctx *fasthttp.RequestCtx, user *cdb.User, db *database.Database, buf *bytes.Buffer) int {
	qp, err := params.ParseQuery(ctx.Request.URI().QueryArgs())
	if err != nil {
		panic(err)
	}

	if len(qp.Params.InfoHashes) == 0 {
		failure("Malformed request - missing info_hash", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	} else if len(qp.Params.InfoHashes) > 1 {
		failure("Malformed request - can only announce singular info_hash", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if len(qp.Params.PeerID) == 0 {
		failure("Malformed request - missing peer_id", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if len(qp.Params.PeerID) != 20 {
		failure("Malformed request - invalid peer_id", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if !qp.Exists.Port {
		failure("Malformed request - missing port", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if strictPort && qp.Params.Port < 1024 || qp.Params.Port > 65535 {
		failure(fmt.Sprintf("Malformed request - port outside of acceptable range (port: %d)", qp.Params.Port),
			buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if !qp.Exists.Uploaded {
		failure("Malformed request - missing uploaded", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if !qp.Exists.Downloaded {
		failure("Malformed request - missing downloaded", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if !qp.Exists.Left {
		failure("Malformed request - missing left", buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	ipAddr := func() string {
		ipV4, existsV4 := getPublicIPV4(qp.Params.IPv4, qp.Exists.IPv4) // First try to get IPv4 address if client sent it
		ip, exists := getPublicIPV4(qp.Params.IP, qp.Exists.IP)         // ... then try to get public IP if sent by client

		// Fail if ip and ipv4 are not same, and both are provided
		if (existsV4 && exists) && (ip != ipV4) {
			return ""
		}

		if existsV4 {
			return ipV4
		}

		if exists {
			return ip
		}

		// Check for proxied IP header
		if proxyHeader, exists := config.Section("http").Get("proxy_header", ""); exists {
			if ips := ctx.Request.Header.PeekAll(proxyHeader); len(ips) > 0 {
				return string(ips[0])
			}
		}

		// Check for IP in socket
		remoteAddrString := ctx.RemoteAddr().String()
		portIndex := strings.LastIndex(remoteAddrString, ":")

		if portIndex != -1 {
			return remoteAddrString[0:portIndex]
		}

		// Everything else failed
		return ""
	}()

	ipBytes := net.ParseIP(ipAddr).To4()
	if nil == ipBytes {
		failure(fmt.Sprintf("Failed to parse IP address (ip: %s)", ipAddr), buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	clientID, matched := clientApproved(qp.Params.PeerID, db)
	if !matched {
		failure(fmt.Sprintf("Your client is not approved (peer_id: %s)", qp.Params.PeerID), buf, 1*time.Hour)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	torrent, exists := (*db.Torrents.Load())[qp.Params.InfoHashes[0]]
	if !exists {
		failure("This torrent does not exist", buf, 5*time.Minute)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	// Take torrent peers lock to read/write on it to prevent race conditions
	torrent.PeerLock()
	defer torrent.PeerUnlock()

	if torrentStatus := torrent.Status.Load(); torrentStatus == 1 && qp.Params.Left == 0 {
		slog.Info("unpruning torrent", "fid", torrent.ID.Load())

		torrent.Status.Store(0)

		/* It is okay to do this asynchronously as tracker's internal in-memory state has already been updated for this
		torrent. While it is technically possible that we will do this more than once in some cases, the state is of
		boolean type so there is no risk of data loss. */
		go db.UnPrune(torrent)
	} else if torrentStatus != 0 {
		failure(fmt.Sprintf("This torrent does not exist (torrentStatus: %d, left: %d)", torrentStatus, qp.Params.Left),
			buf, 15*time.Minute)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	if !qp.Exists.NumWant {
		qp.Params.NumWant = uint16(defaultNumWant)
	} else if qp.Params.NumWant > uint16(maxNumWant) {
		qp.Params.NumWant = uint16(maxNumWant)
	}

	var (
		peer    *cdb.Peer
		peerKey = cdb.NewPeerKey(user.ID.Load(), cdb.PeerIDFromRawString(qp.Params.PeerID))

		now = time.Now().Unix()

		newPeer = false
		seeding = false
		active  = true

		completed = qp.Params.Event == "completed"
	)

	if qp.Params.Left > 0 {
		if isDisabledDownload(db, user, torrent) {
			failure("Your download privileges are disabled", buf, 1*time.Hour)
			return fasthttp.StatusOK // Required by torrent clients to interpret failure response
		}

		peer, exists = torrent.Leechers[peerKey]
		if !exists {
			newPeer = true
			peer = &cdb.Peer{}

			torrent.Leechers[peerKey] = peer
			torrent.LeechersLength.Store(uint32(len(torrent.Leechers)))
		}
	} else if completed {
		peer, exists = torrent.Leechers[peerKey]
		if !exists {
			newPeer = true
			peer = &cdb.Peer{}

			torrent.Seeders[peerKey] = peer
			torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
		} else {
			// Previously tracked peer is now a seeder
			torrent.Seeders[peerKey] = peer
			delete(torrent.Leechers, peerKey)

			torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
			torrent.LeechersLength.Store(uint32(len(torrent.Leechers)))
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
				torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
			} else {
				/* Previously tracked peer is now a seeder, however we never received their "completed" event.
				Broken client? Unreported snatch? Cross-seeding? Let's not report it as snatch to avoid
				over-reporting for cross-seeding */
				torrent.Seeders[peerKey] = peer
				delete(torrent.Leechers, peerKey)

				torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
				torrent.LeechersLength.Store(uint32(len(torrent.Leechers)))
			}
		}
		seeding = true
	}

	// Update peer info/stats
	if newPeer {
		peer.ID = peerKey.PeerID()
		peer.UserID = user.ID.Load()
		peer.TorrentID = torrent.ID.Load()
		peer.StartTime = now
		peer.LastAnnounce = now
		peer.Uploaded = qp.Params.Uploaded
		peer.Downloaded = qp.Params.Downloaded
	}

	peer.Addr = cdb.NewPeerAddressFromIPPort(ipBytes, qp.Params.Port)
	peer.ClientID = clientID

	// If a user restarts a torrent, their delta may be negative, attenuating this to 0 should be fine for stats purposes
	rawDeltaUpload := int64(qp.Params.Uploaded) - int64(peer.Uploaded)
	if rawDeltaUpload < 0 {
		rawDeltaUpload = 0
	}

	rawDeltaDownload := int64(qp.Params.Downloaded) - int64(peer.Downloaded)
	if rawDeltaDownload < 0 {
		rawDeltaDownload = 0
	}

	var (
		torrentGroupDownMultiplier = 1.0
		torrentGroupUpMultiplier   = 1.0
	)

	if torrentGroupFreeleech, exists := (*db.TorrentGroupFreeleech.Load())[torrent.Group.Key()]; exists {
		torrentGroupDownMultiplier = torrentGroupFreeleech.DownMultiplier
		torrentGroupUpMultiplier = torrentGroupFreeleech.UpMultiplier
	}

	var deltaDownload int64
	if !database.GlobalFreeleech.Load() {
		deltaDownload = int64(
			float64(rawDeltaDownload) *
				math.Abs(math.Float64frombits(user.DownMultiplier.Load())) *
				math.Abs(torrentGroupDownMultiplier) *
				math.Abs(math.Float64frombits(torrent.DownMultiplier.Load())),
		)
	}

	deltaUpload := int64(
		float64(rawDeltaUpload) *
			math.Abs(math.Float64frombits(user.UpMultiplier.Load())) *
			math.Abs(torrentGroupUpMultiplier) *
			math.Abs(math.Float64frombits(torrent.UpMultiplier.Load())),
	)

	peer.Uploaded = qp.Params.Uploaded
	peer.Downloaded = qp.Params.Downloaded
	peer.Left = qp.Params.Left
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
		torrent.LastAction.Store(now)
	}

	var deltaSnatch uint8

	if qp.Params.Event == "stopped" {
		/* We can remove the peer from the list and still have their stats be recorded,
		since we still have a reference to their object. After flushing, all references
		should be gone, allowing the peer to be GC'd. */
		if seeding {
			delete(torrent.Seeders, peerKey)
			torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
		} else {
			delete(torrent.Leechers, peerKey)
			torrent.LeechersLength.Store(uint32(len(torrent.Leechers)))
		}

		active = false
	} else if completed {
		deltaSnatch = 1

		db.QueueSnatch(peer, now) // Non-blocking
	}

	persistAddr := peer.Addr // This is done here so that we don't have to keep two instances of Addr for each Peer
	if user.TrackerHide.Load() {
		persistAddr = cdb.NewPeerAddressFromIPPort(net.IP{127, 0, 0, 1}, qp.Params.Port)
	}

	// Underlying queue operations are non-blocking by spawning new goroutine if channel is already full
	db.QueueTorrent(torrent, deltaSnatch)
	db.QueueTransferHistory(peer, rawDeltaUpload, rawDeltaDownload, deltaTime, deltaSeedTime, deltaSnatch, active)
	db.QueueUser(user, rawDeltaUpload, rawDeltaDownload, deltaUpload, deltaDownload)
	db.QueueTransferIP(peer, persistAddr, rawDeltaUpload, rawDeltaDownload)

	// Record must be done in separate goroutine for now; todo: rewrite this so it doesn't tank performance
	go record.Record(peer.TorrentID, user.ID.Load(), peer.Addr, qp.Params.Event, qp.Params.Uploaded,
		qp.Params.Downloaded, qp.Params.Left)

	// Generate response
	seedCount := int(torrent.SeedersLength.Load())
	leechCount := int(torrent.LeechersLength.Load())
	snatchCount := uint16(torrent.Snatched.Load())

	response := make(map[string]interface{})
	response["complete"] = seedCount
	response["incomplete"] = leechCount
	response["downloaded"] = snatchCount
	response["min interval"] = minAnnounceInterval

	/* We ask clients to announce each interval seconds. In order to spread the load on tracker,
	we will vary the interval given to client by random number of seconds between 0 and value
	specified in config */
	announceDrift := util.UnsafeRand(0, maxAccounceDrift)
	response["interval"] = announceInterval + announceDrift

	if qp.Params.NumWant > 0 && active {
		compact := !qp.Exists.Compact || !qp.Params.Compact
		noPeerID := qp.Exists.NoPeerID && qp.Params.NoPeerID

		var peerCount int
		if seeding {
			peerCount = min(int(qp.Params.NumWant), leechCount)
		} else {
			peerCount = min(int(qp.Params.NumWant), leechCount+seedCount-1)
		}

		peersToSend := make([]*cdb.Peer, 0, peerCount)

		/*
		 * The iteration is already "random", so we don't need to randomize ourselves:
		 * - Each time an element is inserted into the map, it gets a some arbitrary position for iteration
		 * - Each time you range over the map, it starts at a random offset into the map's elements
		 */
		if seeding {
			for _, leech := range torrent.Leechers {
				if len(peersToSend) >= int(qp.Params.NumWant) {
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
				if len(peersToSend) >= int(qp.Params.NumWant) {
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
				if len(peersToSend) >= int(qp.Params.NumWant) {
					break
				}
				if leech.UserID == peer.UserID {
					continue
				}
				peersToSend = append(peersToSend, leech)
			}
		}

		if compact {
			peerBuff := make([]byte, 0, len(peersToSend)*cdb.PeerAddressSize)

			for _, other := range peersToSend {
				peerBuff = append(peerBuff, other.Addr[:]...)
			}

			response["peers"] = peerBuff
		} else {
			peerList := make([]map[string]interface{}, len(peersToSend))
			for i, other := range peersToSend {
				peerMap := make(map[string]interface{})
				peerMap["ip"] = other.Addr.IPString()
				peerMap["port"] = other.Addr.Port()
				if !noPeerID {
					peerMap["peer id"] = other.ID[:]
				}
				peerList[i] = peerMap
			}
			response["peers"] = peerList
		}
	}

	encoder := bencode.NewEncoder(buf)
	if err := encoder.Encode(response); err != nil {
		panic(err)
	}

	return fasthttp.StatusOK
}

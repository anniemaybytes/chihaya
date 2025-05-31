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
	"net"
	"net/netip"
	"time"

	"chihaya/database"
	cdb "chihaya/database/types"
	"chihaya/util"

	"github.com/valyala/fasthttp"
)

func failure(err string, buf *bytes.Buffer, interval time.Duration) {
	// Reset buffer to prevent reuse of any written bytes
	buf.Reset()
	util.BencodeFailure(buf, err, interval)
}

func isClientApproved(peerID string, db *database.Database) (uint16, bool) {
	var (
		widLen, i int
		matched   bool
	)

	for id, clientID := range *db.Clients.Load() {
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

func isPasskeyValid(passkey string, db *database.Database) *cdb.User {
	user, exists := (*db.Users.Load())[passkey]
	if !exists {
		return nil
	}

	return user
}

func hasHitAndRun(db *database.Database, userID, torrentID uint32) bool {
	hnr := cdb.UserTorrentPair{
		UserID:    userID,
		TorrentID: torrentID,
	}

	_, exists := (*db.HitAndRuns.Load())[hnr]

	return exists
}

func isDisabledDownload(db *database.Database, user *cdb.User, torrent *cdb.Torrent) bool {
	// Only disable download if the torrent doesn't have a HnR against it
	return user.DisableDownload.Load() && !hasHitAndRun(db, user.ID.Load(), torrent.ID.Load())
}

func isPrivateIPAddress(address netip.Addr) bool {
	return !address.IsGlobalUnicast() || address.IsPrivate()
}

func getIPAddressFromRequest(ctx *fasthttp.RequestCtx) netip.Addr {
	xRealIP := ctx.Request.Header.Peek("X-Real-Ip")
	xForwardedFor := ctx.Request.Header.Peek("X-Forwarded-For")

	// Try to use value from X-Real-Ip header if exists
	if len(xRealIP) > 0 {
		return netip.MustParseAddr(string(xRealIP))
	}

	// Check list of IPs in X-Forwarded-For and try to return the first public address
	for _, remoteBytes := range bytes.Split(xForwardedFor, []byte(",")) {
		if remoteIP, err := netip.ParseAddr(string(bytes.TrimSpace(remoteBytes))); err != nil {
			if !isPrivateIPAddress(remoteIP) {
				return remoteIP
			}
		}
	}

	// Try to use socket address directly
	if addr, ok := ctx.RemoteAddr().(*net.TCPAddr); ok {
		return addr.AddrPort().Addr()
	}

	// Parse address from context (fallback)
	return netip.MustParseAddrPort(ctx.RemoteAddr().String()).Addr()
}

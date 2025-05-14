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
	"errors"
	"net"
	"strings"
	"time"

	"chihaya/database"
	cdb "chihaya/database/types"
	"chihaya/util"

	"github.com/valyala/fasthttp"
)

var (
	cidrs       []*net.IPNet
	errNetParse = errors.New("failed to parse address")
)

func init() {
	maxCidrBlocks := []string{
		"127.0.0.1/8",    // localhost
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"100.64.0.0/10",  // RFC6598
		"::1/128",        // localhost
		"fc00::/7",       // RFC4193
		"fe80::/10",      // RFC5156 link-scoped
	}

	cidrs = make([]*net.IPNet, len(maxCidrBlocks))

	for i, maxCidrBlock := range maxCidrBlocks {
		_, cidr, _ := net.ParseCIDR(maxCidrBlock)
		cidrs[i] = cidr
	}
}

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

func isPrivateIPAddress(address string) (bool, error) {
	ipAddress := net.ParseIP(address)
	if ipAddress == nil { // Invalid IP provided, fail
		return false, errNetParse
	}

	for i := range cidrs {
		if cidrs[i].Contains(ipAddress) {
			return true, nil
		}
	}

	return false, nil
}

func getIPAddressFromRequest(ctx *fasthttp.RequestCtx) (string, error) {
	xRealIP := string(ctx.Request.Header.Peek("X-Real-Ip"))
	xForwardedFor := string(ctx.Request.Header.Peek("X-Forwarded-For"))
	socketAddr := ctx.RemoteAddr().String()

	// Try to use value from X-Real-Ip header if exists
	if len(xRealIP) > 0 {
		return xRealIP, nil
	}

	// Check list of IPs in X-Forwarded-For and try to return the first public address
	for _, remoteIP := range strings.Split(xForwardedFor, ",") {
		remoteIP = strings.TrimSpace(remoteIP)

		isPrivate, err := isPrivateIPAddress(remoteIP)
		if err == nil && !isPrivate {
			return remoteIP, nil
		}
	}

	// Use socket address
	remoteIP, _, err := net.SplitHostPort(socketAddr)
	if err != nil {
		return "", err
	}

	return remoteIP, nil
}

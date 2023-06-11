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
	"context"
	"time"

	"chihaya/database"
	cdb "chihaya/database/types"
	"chihaya/util"

	"github.com/zeebo/bencode"
)

func failure(err string, buf *bytes.Buffer, interval time.Duration) {
	data := make(map[string]interface{})
	data["failure reason"] = err
	data["interval"] = interval / time.Second     // Assuming in seconds
	data["min interval"] = interval / time.Second // Assuming in seconds

	buf.Reset()

	encoder := bencode.NewEncoder(buf)
	if errz := encoder.Encode(buf); errz != nil {
		panic(errz)
	}
}

func clientApproved(peerID string, db *database.Database) (uint16, bool) {
	util.TakeSemaphore(db.ClientsSemaphore)
	defer util.ReturnSemaphore(db.ClientsSemaphore)

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

func isPasskeyValid(ctx context.Context, passkey string, db *database.Database) (*cdb.User, error) {
	if !util.TryTakeSemaphore(ctx, db.UsersSemaphore) {
		return nil, ctx.Err()
	}
	defer util.ReturnSemaphore(db.UsersSemaphore)

	user, exists := db.Users[passkey]
	if !exists {
		return nil, nil
	}

	return user, nil
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
	return user.DisableDownload && !hasHitAndRun(db, user.ID, torrent.ID)
}

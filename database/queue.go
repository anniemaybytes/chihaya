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

package database

import (
	"strconv"

	cdb "chihaya/database/types"
	"chihaya/util"
)

/*
 * For these, we assume that the caller already has a read lock on the record
 *
 * Buffers are used for efficient string concatenation
 * It may look ugly with all the explicit type conversions, but this tracker is about speed
 *
 * These functions take from the buffer pool but don't give back,
 * so it's expected that the buffers are returned in the flush functions
 */

func (db *Database) QueueTorrent(torrent *cdb.Torrent, deltaSnatch uint8) {
	tq := db.bufferPool.Take()

	tq.WriteString("(")
	tq.WriteString(strconv.FormatUint(uint64(torrent.ID.Load()), 10))
	tq.WriteString(",")
	tq.WriteString(strconv.FormatUint(uint64(deltaSnatch), 10))
	tq.WriteString(",")
	tq.WriteString(strconv.FormatUint(uint64(torrent.SeedersLength.Load()), 10))
	tq.WriteString(",")
	tq.WriteString(strconv.FormatUint(uint64(torrent.LeechersLength.Load()), 10))
	tq.WriteString(",")
	tq.WriteString(strconv.FormatInt(torrent.LastAction.Load(), 10))
	tq.WriteString(")")

	select {
	case db.torrentChannel <- tq:
	default:
		go func() {
			db.torrentChannel <- tq
		}()
	}
}

func (db *Database) QueueUser(user *cdb.User, rawDeltaUp, rawDeltaDown, deltaUp, deltaDown int64) {
	uq := db.bufferPool.Take()

	uq.WriteString("(")
	uq.WriteString(strconv.FormatUint(uint64(user.ID.Load()), 10))
	uq.WriteString(",")
	uq.WriteString(strconv.FormatInt(deltaUp, 10))
	uq.WriteString(",")
	uq.WriteString(strconv.FormatInt(deltaDown, 10))
	uq.WriteString(",")
	uq.WriteString(strconv.FormatInt(rawDeltaDown, 10))
	uq.WriteString(",")
	uq.WriteString(strconv.FormatInt(rawDeltaUp, 10))
	uq.WriteString(")")

	select {
	case db.userChannel <- uq:
	default:
		go func() {
			db.userChannel <- uq
		}()
	}
}

func (db *Database) QueueTransferHistory(peer *cdb.Peer, rawDeltaUp, rawDeltaDown, deltaTime, deltaSeedTime int64,
	deltaSnatch uint8, active bool) {
	th := db.bufferPool.Take()

	th.WriteString("(")
	th.WriteString(strconv.FormatUint(uint64(peer.UserID), 10))
	th.WriteString(",")
	th.WriteString(strconv.FormatUint(uint64(peer.TorrentID), 10))
	th.WriteString(",")
	th.WriteString(strconv.FormatInt(rawDeltaUp, 10))
	th.WriteString(",")
	th.WriteString(strconv.FormatInt(rawDeltaDown, 10))
	th.WriteString(",")
	th.WriteString(util.Btoa(peer.Seeding))
	th.WriteString(",")
	th.WriteString(strconv.FormatInt(peer.StartTime, 10))
	th.WriteString(",")
	th.WriteString(strconv.FormatInt(peer.LastAnnounce, 10))
	th.WriteString(",")
	th.WriteString(strconv.FormatInt(deltaTime, 10))
	th.WriteString(",")
	th.WriteString(strconv.FormatInt(deltaSeedTime, 10))
	th.WriteString(",")
	th.WriteString(util.Btoa(active))
	th.WriteString(",")
	th.WriteString(strconv.FormatUint(uint64(deltaSnatch), 10))
	th.WriteString(",")
	th.WriteString(strconv.FormatUint(peer.Left, 10))
	th.WriteString(")")

	select {
	case db.transferHistoryChannel <- th:
	default:
		go func() {
			db.transferHistoryChannel <- th
		}()
	}
}

func (db *Database) QueueTransferIP(peer *cdb.Peer, rawDeltaUp, rawDeltaDown int64) {
	ti := db.bufferPool.Take()

	ti.WriteString("(")
	ti.WriteString(strconv.FormatUint(uint64(peer.UserID), 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatUint(uint64(peer.TorrentID), 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatUint(uint64(peer.ClientID), 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatUint(uint64(peer.Addr.IPNumeric()), 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatUint(uint64(peer.Addr.Port()), 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatInt(rawDeltaUp, 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatInt(rawDeltaDown, 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatInt(peer.StartTime, 10))
	ti.WriteString(",")
	ti.WriteString(strconv.FormatInt(peer.LastAnnounce, 10))
	ti.WriteString(")")

	select {
	case db.transferIpsChannel <- ti:
	default:
		go func() {
			db.transferIpsChannel <- ti
		}()
	}
}

func (db *Database) QueueSnatch(peer *cdb.Peer, now int64) {
	sn := db.bufferPool.Take()

	sn.WriteString("(")
	sn.WriteString(strconv.FormatUint(uint64(peer.UserID), 10))
	sn.WriteString(",")
	sn.WriteString(strconv.FormatUint(uint64(peer.TorrentID), 10))
	sn.WriteString(",")
	sn.WriteString(strconv.FormatInt(now, 10))
	sn.WriteString(")")

	select {
	case db.snatchChannel <- sn:
	default:
		go func() {
			db.snatchChannel <- sn
		}()
	}
}

func (db *Database) UnPrune(torrent *cdb.Torrent) {
	db.mainConn.mutex.Lock()
	db.mainConn.execute(db.unPruneTorrentStmt, torrent.ID.Load())
	db.mainConn.mutex.Unlock()
}

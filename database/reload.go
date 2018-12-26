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
	"chihaya/config"
	"io"
	"log"
	"time"
)

/*
 * Reloading is performed synchronously for each cache to lower database thrashing.
 *
 * Cache synchronization is handled by using sync.RWMutex, which has a bunch of advantages:
 *   - The number of simultaneous readers is arbitrarily high
 *   - Writing is blocked until all current readers release the mutex
 *   - Once a writer locks the mutex, new readers block until the writer unlocks it
 */
func (db *Database) startReloading() {
	go func() {
		count := 0
		for !db.terminate {
			db.waitGroup.Add(1)
			db.loadUsers()
			db.loadHitAndRuns()
			db.loadTorrents()
			db.loadConfig()

			if count%10 == 0 {
				db.loadWhitelist()
			}

			count++
			db.waitGroup.Done()
			time.Sleep(config.DatabaseReloadInterval)
		}
	}()
}

func (db *Database) loadUsers() {
	var err error
	var count uint

	db.UsersMutex.Lock()
	db.mainConn.mutex.Lock()
	start := time.Now()
	result := db.mainConn.query(db.loadUsersStmt)

	newUsers := make(map[string]*User, len(db.Users))

	row := &rowWrapper{result.MakeRow()}

	id := result.Map("ID")
	torrentPass := result.Map("torrent_pass")
	downMultiplier := result.Map("DownMultiplier")
	upMultiplier := result.Map("UpMultiplier")
	disableDownload := result.Map("DisableDownload")
	trackerHide := result.Map("TrackerHide")

	for {
		err = result.ScanRow(row.r)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Panicf("Error scanning user rows: %v", err)
		}

		torrentPass := row.Str(torrentPass)

		old, exists := db.Users[torrentPass]
		if exists && old != nil {
			old.Id = row.Uint64(id)
			old.DownMultiplier = row.Float64(downMultiplier)
			old.UpMultiplier = row.Float64(upMultiplier)
			old.DisableDownload = row.Bool(disableDownload)
			old.TrackerHide = row.Bool(trackerHide)
			newUsers[torrentPass] = old
		} else {
			newUsers[torrentPass] = &User{
				Id:              row.Uint64(id),
				UpMultiplier:    row.Float64(downMultiplier),
				DownMultiplier:  row.Float64(upMultiplier),
				DisableDownload: row.Bool(disableDownload),
				TrackerHide:     row.Bool(trackerHide),
			}
		}
		count++
	}
	db.mainConn.mutex.Unlock()

	db.Users = newUsers
	db.UsersMutex.Unlock()

	log.Printf("User load complete (%d rows, %dms)", count, time.Now().Sub(start).Nanoseconds()/1000000)
}

func (db *Database) loadHitAndRuns() {
	var err error
	var count uint

	db.mainConn.mutex.Lock()
	start := time.Now()
	result := db.mainConn.query(db.loadHnrStmt)

	newHnr := make(map[UserTorrentPair]struct{})

	row := &rowWrapper{result.MakeRow()}

	uid := result.Map("uid")
	fid := result.Map("fid")

	for {
		err = result.ScanRow(row.r)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Panicf("Error scanning hit and run rows: %v", err)
		}

		hnr := UserTorrentPair{
			UserId:    row.Uint64(uid),
			TorrentId: row.Uint64(fid),
		}
		newHnr[hnr] = struct{}{}

		count++
	}
	db.mainConn.mutex.Unlock()

	db.HitAndRuns = newHnr

	log.Printf("Hit and run load complete (%d rows, %dms)", count, time.Now().Sub(start).Nanoseconds()/1000000)
}

func (db *Database) loadTorrents() {
	var err error
	var count uint

	db.TorrentsMutex.Lock()
	db.mainConn.mutex.Lock()
	start := time.Now()
	result := db.mainConn.query(db.loadTorrentsStmt)

	newTorrents := make(map[string]*Torrent)

	row := &rowWrapper{result.MakeRow()}

	id := result.Map("ID")
	infoHash := result.Map("info_hash")
	downMultiplier := result.Map("DownMultiplier")
	upMultiplier := result.Map("UpMultiplier")
	snatched := result.Map("Snatched")
	status := result.Map("Status")

	for {
		err = result.ScanRow(row.r)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Panicf("Error scanning torrent rows: %v", err)
		}

		infoHash := row.Str(infoHash)

		old, exists := db.Torrents[infoHash]
		if exists && old != nil {
			old.Id = row.Uint64(id)
			old.DownMultiplier = row.Float64(downMultiplier)
			old.UpMultiplier = row.Float64(upMultiplier)
			old.Snatched = row.Uint(snatched)
			old.Status = row.Int64(status)
			newTorrents[infoHash] = old
		} else {
			newTorrents[infoHash] = &Torrent{
				Id:             row.Uint64(id),
				UpMultiplier:   row.Float64(upMultiplier),
				DownMultiplier: row.Float64(downMultiplier),
				Snatched:       row.Uint(snatched),
				Status:         row.Int64(status),

				Seeders:  make(map[string]*Peer),
				Leechers: make(map[string]*Peer),
			}
		}
		count++
	}
	db.mainConn.mutex.Unlock()

	db.Torrents = newTorrents
	db.TorrentsMutex.Unlock()

	log.Printf("Torrent load complete (%d rows, %dms)", count, time.Now().Sub(start).Nanoseconds()/1000000)
}

func (db *Database) loadConfig() {
	db.mainConn.mutex.Lock()
	result := db.mainConn.query(db.loadFreeleechStmt)
	for {
		row, err := result.GetRow()
		if err != nil || row == nil {
			break
		} else {
			config.GlobalFreeleech = row.Bool(0)
		}
	}
	db.mainConn.mutex.Unlock()
}

func (db *Database) loadWhitelist() {
	db.WhitelistMutex.Lock()
	db.mainConn.mutex.Lock()
	start := time.Now()
	result := db.mainConn.query(db.loadWhitelistStmt)

	row := result.MakeRow()

	id := result.Map("id")
	peer_id := result.Map("peer_id")

	db.Whitelist = make(map[uint32]string)

	for {
		err := result.ScanRow(row)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Panicf("Error scanning whitelist rows: %v", err)
		}
		db.Whitelist[uint32(row.Uint64(id))] = row.Str(peer_id)
	}
	db.mainConn.mutex.Unlock()
	db.WhitelistMutex.Unlock()

	log.Printf("Whitelist load complete (%d rows, %dms)", len(db.Whitelist), time.Now().Sub(start).Nanoseconds()/1000000)
}

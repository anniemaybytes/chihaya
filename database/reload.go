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
	"chihaya/collectors"
	"chihaya/config"
	"chihaya/database/types"
	"chihaya/log"
	"time"
)

var (
	reloadInterval  int
	GlobalFreeleech = false
)

func init() {
	intervals := config.Section("intervals")

	reloadInterval, _ = intervals.GetInt("database_reload", 45)
}

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
		time.Sleep(time.Duration(reloadInterval) * time.Second)

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
			time.Sleep(time.Duration(reloadInterval) * time.Second)
		}
	}()
}

func (db *Database) loadUsers() {
	var err error

	var count uint

	db.UsersMutex.Lock()
	db.mainConn.mutex.Lock()

	start := time.Now()
	newUsers := make(map[string]*types.User, len(db.Users))
	rows := db.mainConn.query(db.loadUsersStmt)

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var id uint32

		var torrentPass string

		var downMultiplier, upMultiplier float64

		var disableDownload, trackerHide bool

		err = rows.Scan(&id, &torrentPass, &downMultiplier, &upMultiplier, &disableDownload, &trackerHide)
		if err != nil {
			log.Error.Printf("Error scanning user rows: %s", err)
			log.WriteStack()
		}

		old, exists := db.Users[torrentPass]
		if exists && old != nil {
			old.ID = id
			old.DownMultiplier = downMultiplier
			old.UpMultiplier = upMultiplier
			old.DisableDownload = disableDownload
			old.TrackerHide = trackerHide
			newUsers[torrentPass] = old
		} else {
			newUsers[torrentPass] = &types.User{
				ID:              id,
				UpMultiplier:    downMultiplier,
				DownMultiplier:  upMultiplier,
				DisableDownload: disableDownload,
				TrackerHide:     trackerHide,
			}
		}
		count++
	}
	db.mainConn.mutex.Unlock()

	db.Users = newUsers
	db.UsersMutex.Unlock()

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("users", elapsedTime)
	log.Info.Printf("User load complete (%d rows, %s)", count, elapsedTime.String())
}

func (db *Database) loadHitAndRuns() {
	var err error

	var count uint

	db.mainConn.mutex.Lock()

	start := time.Now()
	newHnr := make(map[types.UserTorrentPair]struct{})
	rows := db.mainConn.query(db.loadHnrStmt)

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var uid, fid uint32

		err = rows.Scan(&uid, &fid)
		if err != nil {
			log.Error.Printf("Error scanning hit and run rows: %s", err)
			log.WriteStack()
		}

		hnr := types.UserTorrentPair{
			UserID:    uid,
			TorrentID: fid,
		}
		newHnr[hnr] = struct{}{}

		count++
	}
	db.mainConn.mutex.Unlock()

	db.HitAndRuns = newHnr

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("hit_and_runs", elapsedTime)
	log.Info.Printf("Hit and run load complete (%d rows, %s)", count, elapsedTime.String())
}

func (db *Database) loadTorrents() {
	var err error

	var count uint

	db.TorrentsMutex.Lock()
	db.mainConn.mutex.Lock()

	start := time.Now()
	newTorrents := make(map[string]*types.Torrent)
	rows := db.mainConn.query(db.loadTorrentsStmt)

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			infoHash                     string
			id                           uint32
			downMultiplier, upMultiplier float64
			snatched                     uint16
			status                       uint8
		)

		err = rows.Scan(&id, &infoHash, &downMultiplier, &upMultiplier, &snatched, &status)
		if err != nil {
			log.Error.Printf("Error scanning torrent rows: %s", err)
			log.WriteStack()
		}

		old, exists := db.Torrents[infoHash]
		if exists && old != nil {
			old.ID = id
			old.DownMultiplier = downMultiplier
			old.UpMultiplier = upMultiplier
			old.Snatched = snatched
			old.Status = status
			newTorrents[infoHash] = old
		} else {
			newTorrents[infoHash] = &types.Torrent{
				ID:             id,
				UpMultiplier:   upMultiplier,
				DownMultiplier: downMultiplier,
				Snatched:       snatched,
				Status:         status,

				Seeders:  make(map[string]*types.Peer),
				Leechers: make(map[string]*types.Peer),
			}
		}
		count++
	}
	db.mainConn.mutex.Unlock()

	db.Torrents = newTorrents
	db.TorrentsMutex.Unlock()

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("torrents", elapsedTime)
	log.Info.Printf("Torrent load complete (%d rows, %s)", count, elapsedTime.String())
}

func (db *Database) loadConfig() {
	db.mainConn.mutex.Lock()
	rows := db.mainConn.query(db.loadFreeleechStmt)

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var globalFreelech bool

		err := rows.Scan(&globalFreelech)
		if err != nil {
			log.Error.Printf("Error scanning torrent rows: %s", err)
			log.WriteStack()
		}

		GlobalFreeleech = globalFreelech
	}
	db.mainConn.mutex.Unlock()
}

func (db *Database) loadWhitelist() {
	db.WhitelistMutex.Lock()
	db.mainConn.mutex.Lock()

	start := time.Now()
	rows := db.mainConn.query(db.loadWhitelistStmt)

	defer func() {
		_ = rows.Close()
	}()

	db.Whitelist = make(map[uint16]string)

	for rows.Next() {
		var (
			id     uint16
			peerID string
		)

		err := rows.Scan(&id, &peerID)
		if err != nil {
			log.Error.Printf("Error scanning whitelist rows: %s", err)
			log.WriteStack()
		}

		db.Whitelist[id] = peerID
	}

	db.mainConn.mutex.Unlock()
	db.WhitelistMutex.Unlock()

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("whitelist", elapsedTime)
	log.Info.Printf("Whitelist load complete (%d rows, %s)", len(db.Whitelist), elapsedTime.String())
}

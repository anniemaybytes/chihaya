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
				db.loadClients()
			}
			count++

			db.waitGroup.Done()
			time.Sleep(time.Duration(reloadInterval) * time.Second)
		}
	}()
}

func (db *Database) loadUsers() {
	db.UsersMutex.Lock()
	db.mainConn.mutex.Lock()

	defer func() {
		db.UsersMutex.Unlock()
		db.mainConn.mutex.Unlock()
	}()

	start := time.Now()
	newUsers := make(map[string]*types.User, len(db.Users))

	rows := db.mainConn.query(db.loadUsersStmt)
	if rows == nil {
		log.Error.Print("Failed to load hit and runs from database")
		log.WriteStack()

		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			id                           uint32
			torrentPass                  string
			downMultiplier, upMultiplier float64
			disableDownload, trackerHide bool
		)

		err := rows.Scan(&id, &torrentPass, &downMultiplier, &upMultiplier, &disableDownload, &trackerHide)
		if err != nil {
			log.Error.Printf("Error scanning user row: %s", err)
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
				UpMultiplier:    upMultiplier,
				DownMultiplier:  downMultiplier,
				DisableDownload: disableDownload,
				TrackerHide:     trackerHide,
			}
		}
	}

	db.Users = newUsers

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("users", elapsedTime)
	log.Info.Printf("User load complete (%d rows, %s)", len(db.Users), elapsedTime.String())
}

func (db *Database) loadHitAndRuns() {
	db.mainConn.mutex.Lock()

	defer func() {
		db.mainConn.mutex.Unlock()
	}()

	start := time.Now()
	newHnr := make(map[types.UserTorrentPair]struct{})

	rows := db.mainConn.query(db.loadHnrStmt)
	if rows == nil {
		log.Error.Print("Failed to load hit and runs from database")
		log.WriteStack()

		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var uid, fid uint32

		err := rows.Scan(&uid, &fid)
		if err != nil {
			log.Error.Printf("Error scanning hit and run row: %s", err)
			log.WriteStack()
		}

		hnr := types.UserTorrentPair{
			UserID:    uid,
			TorrentID: fid,
		}
		newHnr[hnr] = struct{}{}
	}

	db.HitAndRuns = newHnr

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("hit_and_runs", elapsedTime)
	log.Info.Printf("Hit and run load complete (%d rows, %s)", len(db.HitAndRuns), elapsedTime.String())
}

func (db *Database) loadTorrents() {
	db.TorrentsMutex.Lock()
	db.mainConn.mutex.Lock()

	defer func() {
		db.TorrentsMutex.Unlock()
		db.mainConn.mutex.Unlock()
	}()

	start := time.Now()
	newTorrents := make(map[string]*types.Torrent)

	rows := db.mainConn.query(db.loadTorrentsStmt)
	if rows == nil {
		log.Error.Print("Failed to load torrents from database")
		log.WriteStack()

		return
	}

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

		err := rows.Scan(&id, &infoHash, &downMultiplier, &upMultiplier, &snatched, &status)
		if err != nil {
			log.Error.Printf("Error scanning torrent row: %s", err)
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
	}

	db.Torrents = newTorrents

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("torrents", elapsedTime)
	log.Info.Printf("Torrent load complete (%d rows, %s)", len(db.Torrents), elapsedTime.String())
}

func (db *Database) loadConfig() {
	db.mainConn.mutex.Lock()

	defer func() {
		db.mainConn.mutex.Unlock()
	}()

	rows := db.mainConn.query(db.loadFreeleechStmt)
	if rows == nil {
		log.Error.Printf("Failed to load config from database")
		log.WriteStack()

		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var globalFreelech bool

		err := rows.Scan(&globalFreelech)
		if err != nil {
			log.Error.Printf("Error scanning config row: %s", err)
			log.WriteStack()
		}

		GlobalFreeleech = globalFreelech
	}
}

func (db *Database) loadClients() {
	db.ClientsMutex.Lock()
	db.mainConn.mutex.Lock()

	defer func() {
		db.ClientsMutex.Unlock()
		db.mainConn.mutex.Unlock()
	}()

	start := time.Now()

	rows := db.mainConn.query(db.loadClientsStmt)
	if rows == nil {
		log.Error.Print("Failed to load clients from database")
		log.WriteStack()

		return
	}

	defer func() {
		_ = rows.Close()
	}()

	newClients := make(map[uint16]string)

	for rows.Next() {
		var (
			id     uint16
			peerID string
		)

		err := rows.Scan(&id, &peerID)
		if err != nil {
			log.Error.Printf("Error scanning client list row: %s", err)
			log.WriteStack()
		}

		newClients[id] = peerID
	}

	db.Clients = newClients

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("clients", elapsedTime)
	log.Info.Printf("Client list load complete (%d rows, %s)", len(db.Clients), elapsedTime.String())
}

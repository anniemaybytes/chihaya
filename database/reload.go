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
	"sync/atomic"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	cdb "chihaya/database/types"
	"chihaya/log"
	"chihaya/util"
)

var (
	reloadInterval int
	// GlobalFreeleech indicates whether site is now in freeleech mode (takes precedence over torrent-specific multipliers)
	GlobalFreeleech atomic.Bool
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
		for !db.terminate {
			time.Sleep(time.Duration(reloadInterval) * time.Second)

			db.waitGroup.Add(1)
			db.loadUsers()
			db.loadHitAndRuns()
			db.loadTorrents()
			db.loadGroupsFreeleech()
			db.loadConfig()
			db.loadClients()
			db.waitGroup.Done()
		}
	}()
}

func (db *Database) loadUsers() {
	util.TakeSemaphore(db.UsersSemaphore)
	defer util.ReturnSemaphore(db.UsersSemaphore)

	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newUsers := make(map[string]*cdb.User, len(db.Users))

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

		if err := rows.Scan(&id, &torrentPass, &downMultiplier, &upMultiplier, &disableDownload, &trackerHide); err != nil {
			log.Error.Printf("Error scanning user row: %s", err)
			log.WriteStack()
		}

		if old, exists := db.Users[torrentPass]; exists && old != nil {
			old.ID = id
			old.DownMultiplier = downMultiplier
			old.UpMultiplier = upMultiplier
			old.DisableDownload = disableDownload
			old.TrackerHide = trackerHide

			newUsers[torrentPass] = old
		} else {
			newUsers[torrentPass] = &cdb.User{
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
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newHnr := make(map[cdb.UserTorrentPair]struct{})

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

		if err := rows.Scan(&uid, &fid); err != nil {
			log.Error.Printf("Error scanning hit and run row: %s", err)
			log.WriteStack()
		}

		hnr := cdb.UserTorrentPair{
			UserID:    uid,
			TorrentID: fid,
		}
		newHnr[hnr] = struct{}{}
	}

	db.HitAndRuns.Store(&newHnr)

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("hit_and_runs", elapsedTime)
	log.Info.Printf("Hit and run load complete (%d rows, %s)", len(newHnr), elapsedTime.String())
}

func (db *Database) loadTorrents() {
	util.TakeSemaphore(db.TorrentsSemaphore)
	defer util.ReturnSemaphore(db.TorrentsSemaphore)

	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newTorrents := make(map[string]*cdb.Torrent)

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
			group                        cdb.TorrentGroup
		)

		if err := rows.Scan(
			&id,
			&infoHash,
			&downMultiplier,
			&upMultiplier,
			&snatched,
			&status,
			&group.GroupID,
			&group.TorrentType,
		); err != nil {
			log.Error.Printf("Error scanning torrent row: %s", err)
			log.WriteStack()
		}

		if old, exists := db.Torrents[infoHash]; exists && old != nil {
			old.ID = id
			old.DownMultiplier = downMultiplier
			old.UpMultiplier = upMultiplier
			old.Snatched = snatched
			old.Status = status
			old.Group = group

			newTorrents[infoHash] = old
		} else {
			newTorrents[infoHash] = &cdb.Torrent{
				ID:             id,
				UpMultiplier:   upMultiplier,
				DownMultiplier: downMultiplier,
				Snatched:       snatched,
				Status:         status,
				Group:          group,

				Seeders:  make(map[string]*cdb.Peer),
				Leechers: make(map[string]*cdb.Peer),
			}
		}
	}

	db.Torrents = newTorrents

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("torrents", elapsedTime)
	log.Info.Printf("Torrent load complete (%d rows, %s)", len(db.Torrents), elapsedTime.String())
}

func (db *Database) loadGroupsFreeleech() {
	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newTorrentGroupFreeleech := make(map[cdb.TorrentGroup]*cdb.TorrentGroupFreeleech)

	rows := db.mainConn.query(db.loadTorrentGroupFreeleechStmt)
	if rows == nil {
		log.Error.Print("Failed to load torrent group freeleech data from database")
		log.WriteStack()

		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			downMultiplier, upMultiplier float64
			group                        cdb.TorrentGroup
		)

		if err := rows.Scan(&group.GroupID, &group.TorrentType, &downMultiplier, &upMultiplier); err != nil {
			log.Error.Printf("Error scanning torrent row: %s", err)
			log.WriteStack()
		}

		newTorrentGroupFreeleech[group] = &cdb.TorrentGroupFreeleech{
			UpMultiplier:   upMultiplier,
			DownMultiplier: downMultiplier,
		}
	}

	db.TorrentGroupFreeleech.Store(&newTorrentGroupFreeleech)

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("groups_freeleech", elapsedTime)
	log.Info.Printf("Group freeleech load complete (%d rows, %s)", len(db.Torrents), elapsedTime.String())
}

func (db *Database) loadConfig() {
	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	rows := db.mainConn.query(db.loadFreeleechStmt)
	if rows == nil {
		log.Error.Print("Failed to load config from database")
		log.WriteStack()

		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var globalFreelech bool

		if err := rows.Scan(&globalFreelech); err != nil {
			log.Error.Printf("Error scanning config row: %s", err)
			log.WriteStack()
		}

		GlobalFreeleech.Store(globalFreelech)
	}
}

func (db *Database) loadClients() {
	util.TakeSemaphore(db.ClientsSemaphore)
	defer util.ReturnSemaphore(db.ClientsSemaphore)

	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newClients := make(map[uint16]string)

	rows := db.mainConn.query(db.loadClientsStmt)
	if rows == nil {
		log.Error.Print("Failed to load clients from database")
		log.WriteStack()

		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			id     uint16
			peerID string
		)

		if err := rows.Scan(&id, &peerID); err != nil {
			log.Error.Printf("Error scanning client list row: %s", err)
			log.WriteStack()
		}

		newClients[id] = peerID
	}

	db.Clients.Store(&newClients)

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("clients", elapsedTime)
	log.Info.Printf("Client list load complete (%d rows, %s)", len(newClients), elapsedTime.String())
}

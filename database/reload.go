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
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	cdb "chihaya/database/types"
)

// GlobalFreeleech indicates whether site is now in freeleech mode (takes precedence over torrent-specific multipliers)
var GlobalFreeleech atomic.Bool

var (
	reloadInterval int
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
	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()

	dbUsers := *db.Users.Load()
	newUsers := make(map[string]*cdb.User, len(dbUsers))

	rows := db.mainConn.query(db.loadUsersStmt)
	if rows == nil {
		slog.Error("failed to reload from database", "source", "users")
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
			slog.Warn("error scanning row", "source", "users", "err", err)
			continue
		}

		if old, exists := dbUsers[torrentPass]; exists && old != nil {
			old.ID.Store(id)
			old.DownMultiplier.Store(math.Float64bits(downMultiplier))
			old.UpMultiplier.Store(math.Float64bits(upMultiplier))
			old.DisableDownload.Store(disableDownload)
			old.TrackerHide.Store(trackerHide)

			newUsers[torrentPass] = old
		} else {
			u := &cdb.User{}
			u.ID.Store(id)
			u.DownMultiplier.Store(math.Float64bits(downMultiplier))
			u.UpMultiplier.Store(math.Float64bits(upMultiplier))
			u.DisableDownload.Store(disableDownload)
			u.TrackerHide.Store(trackerHide)
			newUsers[torrentPass] = u
		}
	}

	db.Users.Store(&newUsers)

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("users", elapsedTime)
	slog.Info("reload from database", "source", "users", "rows", len(newUsers), "elapsed", elapsedTime)
}

func (db *Database) loadHitAndRuns() {
	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newHnr := make(map[cdb.UserTorrentPair]struct{})

	rows := db.mainConn.query(db.loadHnrStmt)
	if rows == nil {
		slog.Error("failed to reload from database", "source", "hit_and_runs")
		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var uid, fid uint32

		if err := rows.Scan(&uid, &fid); err != nil {
			slog.Warn("error scanning row", "source", "hit_and_runs", "err", err)
			continue
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
	slog.Info("reload from database", "source", "hit_and_runs", "rows", len(newHnr), "elapsed", elapsedTime)
}

func (db *Database) loadTorrents() {
	var start time.Time

	dbTorrents := *db.Torrents.Load()

	newTorrents := make(map[cdb.TorrentHash]*cdb.Torrent, len(dbTorrents))

	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start = time.Now()

	rows := db.mainConn.query(db.loadTorrentsStmt)
	if rows == nil {
		slog.Error("failed to reload from database", "torrents", "hit_and_runs")
		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			infoHash                     cdb.TorrentHash
			id                           uint32
			downMultiplier, upMultiplier float64
			snatched                     uint16
			status                       uint8
			groupID                      uint32
			torrentType                  string
		)

		if err := rows.Scan(
			&id,
			&infoHash,
			&downMultiplier,
			&upMultiplier,
			&snatched,
			&status,
			&groupID,
			&torrentType,
		); err != nil {
			slog.Warn("error scanning row", "source", "torrents", "err", err)
			continue
		}

		torrentTypeUint64, err := cdb.TorrentTypeFromString(torrentType)
		if err != nil {
			slog.Warn("error storing row", "source", "torrents", "err", err)
			continue
		}

		if old, exists := dbTorrents[infoHash]; exists && old != nil {
			old.ID.Store(id)
			old.DownMultiplier.Store(math.Float64bits(downMultiplier))
			old.UpMultiplier.Store(math.Float64bits(upMultiplier))
			old.Snatched.Store(uint32(snatched))
			old.Status.Store(uint32(status))

			old.Group.TorrentType.Store(torrentTypeUint64)
			old.Group.GroupID.Store(groupID)

			newTorrents[infoHash] = old
		} else {
			t := &cdb.Torrent{
				Seeders:  make(map[cdb.PeerKey]*cdb.Peer),
				Leechers: make(map[cdb.PeerKey]*cdb.Peer),
			}

			t.ID.Store(id)
			t.DownMultiplier.Store(math.Float64bits(downMultiplier))
			t.UpMultiplier.Store(math.Float64bits(upMultiplier))
			t.Snatched.Store(uint32(snatched))
			t.Status.Store(uint32(status))

			t.Group.TorrentType.Store(torrentTypeUint64)
			t.Group.GroupID.Store(groupID)

			newTorrents[infoHash] = t
		}
	}

	db.Torrents.Store(&newTorrents)

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("torrents", elapsedTime)
	slog.Info("reload from database", "source", "torrents", "rows", len(newTorrents), "elapsed", elapsedTime)
}

func (db *Database) loadGroupsFreeleech() {
	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newTorrentGroupFreeleech := make(map[cdb.TorrentGroupKey]*cdb.TorrentGroupFreeleech)

	rows := db.mainConn.query(db.loadTorrentGroupFreeleechStmt)
	if rows == nil {
		slog.Error("failed to reload from database", "torrents_group_freeleech", "hit_and_runs")
		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			downMultiplier, upMultiplier float64
			groupID                      uint32
			torrentType                  string
		)

		if err := rows.Scan(&groupID, &torrentType, &downMultiplier, &upMultiplier); err != nil {
			slog.Warn("error scanning row", "source", "torrents_group_freeleech", "err", err)
			continue
		}

		k, err := cdb.TorrentGroupKeyFromString(torrentType, groupID)
		if err != nil {
			slog.Warn("error storing row", "source", "torrents_group_freeleech", "err", err)
			continue
		}

		newTorrentGroupFreeleech[k] = &cdb.TorrentGroupFreeleech{
			UpMultiplier:   upMultiplier,
			DownMultiplier: downMultiplier,
		}
	}

	db.TorrentGroupFreeleech.Store(&newTorrentGroupFreeleech)

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("groups_freeleech", elapsedTime)
	slog.Info("reload from database", "source", "torrents_group_freeleech",
		"rows", len(newTorrentGroupFreeleech), "elapsed", elapsedTime)
}

func (db *Database) loadConfig() {
	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	rows := db.mainConn.query(db.loadFreeleechStmt)
	if rows == nil {
		slog.Error("failed to reload from database", "source", "config")
		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var globalFreelech bool

		if err := rows.Scan(&globalFreelech); err != nil {
			slog.Warn("error scanning row", "source", "config", "err", err)
			continue
		}

		GlobalFreeleech.Store(globalFreelech)
	}
}

func (db *Database) loadClients() {
	db.mainConn.mutex.Lock()
	defer db.mainConn.mutex.Unlock()

	start := time.Now()
	newClients := make(map[uint16]string)

	rows := db.mainConn.query(db.loadClientsStmt)
	if rows == nil {
		slog.Error("failed to reload from database", "source", "approved_clients")
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
			slog.Warn("error scanning row", "source", "approved_clients", "err", err)
		}

		newClients[id] = peerID
	}

	db.Clients.Store(&newClients)

	elapsedTime := time.Since(start)
	collectors.UpdateReloadTime("clients", elapsedTime)
	slog.Info("reload from database", "source", "approved_clients", "rows", len(newClients), "elapsed", elapsedTime)
}

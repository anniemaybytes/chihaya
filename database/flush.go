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
	"bytes"
	"chihaya/collectors"
	"chihaya/config"
	"chihaya/log"
	"chihaya/util"
	"time"
)

var (
	peerInactivityInterval     int
	purgeInactivePeersInterval int
	flushSleepInterval         int
)

func init() {
	intervals := config.Section("intervals")

	peerInactivityInterval, _ = intervals.GetInt("peer_inactivity", 3900)
	purgeInactivePeersInterval, _ = intervals.GetInt("purge_inactive_peers", 120)

	result, exists := intervals.GetInt("flush", 5)
	if !exists {
		log.Warning.Println("FlushSleepInterval is undefined, using default of 5 seconds - this WILL affect performance!")
	}

	flushSleepInterval = result
}

/*
 * Channels are used for flushing to limit throughput to a manageable level.
 * If a client causes an update that requires a flush, it writes to the channel requesting that a flush occur.
 * However, if the channel is already full (to xFlushBufferSize), the client thread blocks until a flush occurs.
 * This way, rather than thrashing and missing flushes, clients are simply forced to wait longer.
 *
 * This tradeoff can be adjusted by tweaking the various xFlushBufferSize values to suit the server.
 *
 * Each flush routine now gets its own database connection to maximize update throughput.
 */

/*
 * If a buffer channel is less than half full on a flush, the routine will wait some time before flushing again.
 * If the channel is more than half full, it doesn't wait at all.
 */

var (
	torrentFlushBufferSize         int
	userFlushBufferSize            int
	transferHistoryFlushBufferSize int
	transferIpsFlushBufferSize     int
	snatchFlushBufferSize          int
)

func (db *Database) startFlushing() {
	db.torrentChannel = make(chan *bytes.Buffer, torrentFlushBufferSize)
	db.userChannel = make(chan *bytes.Buffer, userFlushBufferSize)
	db.transferHistoryChannel = make(chan *bytes.Buffer, transferHistoryFlushBufferSize)
	db.transferIpsChannel = make(chan *bytes.Buffer, transferIpsFlushBufferSize)
	db.snatchChannel = make(chan *bytes.Buffer, snatchFlushBufferSize)

	go db.flushTorrents()
	go db.flushUsers()
	go db.flushTransferHistory() // Can not be blocking or it will lock purgeInactivePeers when chan is empty
	go db.flushTransferIps()
	go db.flushSnatches()

	go db.purgeInactivePeers()
}

func (db *Database) flushTorrents() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	conn := Open()

	for {
		length := util.Max(1, len(db.torrentChannel))

		query.Reset()
		query.WriteString("CREATE TEMPORARY TABLE IF NOT EXISTS flush_torrents (" +
			"ID int unsigned NOT NULL, " +
			"Snatched int unsigned NOT NULL DEFAULT 0, " +
			"Seeders int unsigned NOT NULL DEFAULT 0, " +
			"Leechers int unsigned NOT NULL DEFAULT 0, " +
			"last_action int NOT NULL DEFAULT 0, " +
			"PRIMARY KEY (ID)) ENGINE=MEMORY")
		conn.exec(&query)

		query.Reset()
		query.WriteString("TRUNCATE flush_torrents")
		conn.exec(&query)

		query.Reset()
		query.WriteString("INSERT INTO flush_torrents VALUES ")

		for count = 0; count < length; count++ {
			b := <-db.torrentChannel
			if b == nil {
				break
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		logFlushes, _ := config.GetBool("log_flushes", true)
		if logFlushes && !db.terminate {
			log.Info.Printf("{torrents} Flushing %d\n", count)
		}

		if count > 0 {
			startTime := time.Now()

			query.WriteString(" ON DUPLICATE KEY UPDATE Snatched = Snatched + VALUE(Snatched), " +
				"Seeders = VALUE(Seeders), Leechers = VALUE(Leechers), " +
				"last_action = IF(last_action < VALUE(last_action), VALUE(last_action), last_action)")
			conn.exec(&query)

			query.Reset()
			query.WriteString("UPDATE torrents t, flush_torrents ft SET " +
				"t.Snatched = t.Snatched + ft.Snatched, " +
				"t.Seeders = ft.Seeders, " +
				"t.Leechers = ft.Leechers, " +
				"t.last_action = IF(t.last_action < ft.last_action, ft.last_action, t.last_action)" +
				"WHERE t.ID = ft.ID")
			conn.exec(&query)

			query.Reset()
			query.WriteString("UPDATE torrents_group tg, torrents t, flush_torrents ft SET " +
				"tg.Time=UNIX_TIMESTAMP() " +
				"WHERE tg.ID = t.GroupID AND t.TorrentType = 'anime' AND t.ID = ft.ID")
			conn.exec(&query)

			query.Reset()
			query.WriteString("UPDATE torrents_group2 tg, torrents t, flush_torrents ft SET " +
				"tg.Time=UNIX_TIMESTAMP() " +
				"WHERE tg.ID = t.GroupID AND t.TorrentType = 'music' AND t.ID = ft.ID")
			conn.exec(&query)

			if !db.terminate {
				elapsedTime := time.Since(startTime)
				collectors.UpdateFlushTime("torrents", elapsedTime)
				collectors.UpdateChannelsLen("torrents", count)
			}

			if length < (torrentFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	_ = conn.Close()
}

func (db *Database) flushUsers() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	conn := Open()

	for {
		length := util.Max(1, len(db.userChannel))

		query.Reset()
		query.WriteString("CREATE TEMPORARY TABLE IF NOT EXISTS flush_users (" +
			"ID int unsigned NOT NULL, " +
			"Uploaded bigint unsigned NOT NULL DEFAULT 0, " +
			"Downloaded bigint unsigned NOT NULL DEFAULT 0, " +
			"rawdl bigint unsigned NOT NULL DEFAULT 0, " +
			"rawup bigint unsigned NOT NULL DEFAULT 0, " +
			"PRIMARY KEY (ID)) ENGINE=MEMORY")
		conn.exec(&query)

		query.Reset()
		query.WriteString("TRUNCATE flush_users")
		conn.exec(&query)

		query.Reset()
		query.WriteString("INSERT INTO flush_users VALUES ")

		for count = 0; count < length; count++ {
			b := <-db.userChannel
			if b == nil {
				break
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		logFlushes, _ := config.GetBool("log_flushes", true)
		if logFlushes && !db.terminate {
			log.Info.Printf("{users_main} Flushing %d\n", count)
		}

		if count > 0 {
			startTime := time.Now()

			query.WriteString(" ON DUPLICATE KEY UPDATE Uploaded = Uploaded + VALUE(Uploaded), " +
				"Downloaded = Downloaded + VALUE(Downloaded), rawdl = rawdl + VALUE(rawdl), rawup = rawup + VALUE(rawup)")
			conn.exec(&query)

			query.Reset()
			query.WriteString("UPDATE users_main u, flush_users fu SET " +
				"u.Uploaded = u.Uploaded + fu.Uploaded, " +
				"u.Downloaded = u.Downloaded + fu.Downloaded, " +
				"u.rawdl = u.rawdl + fu.rawdl, " +
				"u.rawup = u.rawup + fu.rawup " +
				"WHERE u.ID = fu.ID")
			conn.exec(&query)

			if !db.terminate {
				elapsedTime := time.Since(startTime)
				collectors.UpdateFlushTime("users", elapsedTime)
				collectors.UpdateChannelsLen("users", count)
			}

			if length < (userFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	_ = conn.Close()
}

func (db *Database) flushTransferHistory() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	conn := Open()

main:
	for {
		db.transferHistoryWaitGroupMu.Lock()
		if db.transferHistoryWaitGroupSe == 1 {
			db.transferHistoryWaitGroupMu.Unlock()
			log.Warning.Printf("goTransferHistoryWait has started... (skipping flushTransferHistory)")
			time.Sleep(time.Second)
			continue
		}
		db.transferHistoryWaitGroup.Add(1)
		db.transferHistoryWaitGroupMu.Unlock()

		length := util.Max(1, len(db.transferHistoryChannel))
		query.Reset()

		query.WriteString("INSERT INTO transfer_history (uid, fid, uploaded, downloaded, " +
			"seeding, starttime, last_announce, activetime, seedtime, active, snatched, remaining) VALUES\n")

	counter:
		for count = 0; count < length; count++ {
			select {
			case b, ok := <-db.transferHistoryChannel:
				if ok {
					query.Write(b.Bytes())
					db.bufferPool.Give(b)

					if count != length-1 {
						query.WriteRune(',')
					}
				} else {
					break counter
				}
			default:
				db.transferHistoryWaitGroup.Done()
				time.Sleep(time.Second)
				continue main
			}
		}

		logFlushes, _ := config.GetBool("log_flushes", true)
		if logFlushes && !db.terminate {
			log.Info.Printf("{transfer_history} Flushing %d\n", count)
		}

		if count > 0 {
			startTime := time.Now()

			query.WriteString("\nON DUPLICATE KEY UPDATE uploaded = uploaded + VALUE(uploaded), " +
				"downloaded = downloaded + VALUE(downloaded), remaining = VALUE(remaining), " +
				"seeding = VALUE(seeding), activetime = activetime + VALUE(activetime), " +
				"seedtime = seedtime + VALUE(seedtime), last_announce = VALUE(last_announce), " +
				"active = VALUE(active), snatched = snatched + VALUE(snatched);")

			conn.exec(&query)
			db.transferHistoryWaitGroup.Done()

			if !db.terminate {
				elapsedTime := time.Since(startTime)
				collectors.UpdateFlushTime("transfer_history", elapsedTime)
				collectors.UpdateChannelsLen("transfer_history", count)
			}

			if length < (transferHistoryFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate {
			db.transferHistoryWaitGroup.Done()
			break main
		} else {
			db.transferHistoryWaitGroup.Done()
			time.Sleep(time.Second)
		}
	}

	_ = conn.Close()
}

func (db *Database) flushTransferIps() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	conn := Open()

	for {
		length := util.Max(1, len(db.transferIpsChannel))

		query.Reset()
		query.WriteString("INSERT INTO transfer_ips (uid, fid, client_id, ip, port, uploaded, downloaded, " +
			"starttime, last_announce) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.transferIpsChannel
			if b == nil {
				break
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		logFlushes, _ := config.GetBool("log_flushes", true)
		if logFlushes && !db.terminate {
			log.Info.Printf("{transfer_ips} Flushing %d\n", count)
		}

		if count > 0 {
			startTime := time.Now()

			// todo in future, port should be part of PK
			query.WriteString("\nON DUPLICATE KEY UPDATE port = VALUE(port), downloaded = downloaded + VALUE(downloaded), " +
				"uploaded = uploaded + VALUE(uploaded), last_announce = VALUE(last_announce)")
			conn.exec(&query)

			if !db.terminate {
				elapsedTime := time.Since(startTime)
				collectors.UpdateFlushTime("transfer_ips", elapsedTime)
				collectors.UpdateChannelsLen("transfer_ips", count)
			}

			if length < (transferIpsFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	_ = conn.Close()
}

func (db *Database) flushSnatches() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	conn := Open()

	for {
		length := util.Max(1, len(db.snatchChannel))

		query.Reset()
		query.WriteString("INSERT INTO transfer_history (uid, fid, snatched_time) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.snatchChannel
			if b == nil {
				break
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		logFlushes, _ := config.GetBool("log_flushes", true)
		if logFlushes && !db.terminate {
			log.Info.Printf("{snatches} Flushing %d\n", count)
		}

		if count > 0 {
			startTime := time.Now()

			query.WriteString("\nON DUPLICATE KEY UPDATE snatched_time = VALUE(snatched_time)")
			conn.exec(&query)

			if !db.terminate {
				elapsedTime := time.Since(startTime)
				collectors.UpdateFlushTime("snatches", elapsedTime)
				collectors.UpdateChannelsLen("snatches", count)
			}

			if length < (snatchFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	_ = conn.Close()
}

func (db *Database) purgeInactivePeers() {
	time.Sleep(2 * time.Second)

	for !db.terminate {
		db.waitGroup.Add(1)

		start := time.Now()
		now := start.Unix()
		count := 0

		oldestActive := now - int64(peerInactivityInterval)

		// First, remove inactive peers from memory
		db.TorrentsMutex.Lock()
		for _, torrent := range db.Torrents {
			countThisTorrent := count

			for id, peer := range torrent.Leechers {
				if peer.LastAnnounce < oldestActive {
					delete(torrent.Leechers, id)
					count++
				}
			}

			for id, peer := range torrent.Seeders {
				if peer.LastAnnounce < oldestActive {
					delete(torrent.Seeders, id)
					count++
				}
			}

			if countThisTorrent != count {
				db.RecordTorrent(torrent, 0)
			}
		}
		db.TorrentsMutex.Unlock()

		elapsedTime := time.Since(start)
		collectors.UpdateFlushTime("purging_inactive_peers", elapsedTime)
		log.Info.Printf("Purged %d inactive peers from memory (%s)\n", count, elapsedTime.String())

		// Wait to prevent a race condition where the user has announced but their announce time hasn't been flushed yet
		db.goTransferHistoryWait()

		// Then set them to inactive in the database
		db.mainConn.mutex.Lock()

		start = time.Now()
		result := db.mainConn.execute(db.cleanStalePeersStmt, oldestActive)

		rows, err := result.RowsAffected()
		if err != nil {
			log.Error.Printf("Error in getting affected rows: %s", err)
			log.WriteStack()
		}
		db.mainConn.mutex.Unlock()

		log.Info.Printf("Updated %d inactive peers in database (%s)\n", rows, time.Since(start).String())

		db.waitGroup.Done()
		time.Sleep(time.Duration(purgeInactivePeersInterval) * time.Second)
	}
}

func (db *Database) goTransferHistoryWait() {
	log.Info.Printf("Starting goTransferHistoryWait")
	db.transferHistoryWaitGroupMu.Lock()
	db.transferHistoryWaitGroupSe = 1
	db.transferHistoryWaitGroupMu.Unlock()
	db.transferHistoryWaitGroup.Wait()
	db.transferHistoryWaitGroupMu.Lock()
	db.transferHistoryWaitGroupSe = 0
	db.transferHistoryWaitGroupMu.Unlock()
	log.Info.Printf("Releasing goTransferHistoryWait")
}

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

func (db *Database) startFlushing() {
	db.torrentChannel = make(chan *bytes.Buffer, config.TorrentFlushBufferSize)
	db.userChannel = make(chan *bytes.Buffer, config.UserFlushBufferSize)
	db.transferHistoryChannel = make(chan *bytes.Buffer, config.TransferHistoryFlushBufferSize)
	db.transferIpsChannel = make(chan *bytes.Buffer, config.TransferIpsFlushBufferSize)
	db.snatchChannel = make(chan *bytes.Buffer, config.SnatchFlushBufferSize)

	go db.flushTorrents()
	go db.flushUsers()
	go db.flushTransferHistory() // this can not be blocking because it will lock purgeInactivePeers from executing when channel is empty
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

	conn := OpenDatabaseConnection()

	for {
		length := util.Max(1, len(db.torrentChannel))

		query.Reset()
		query.WriteString("DROP TABLE IF EXISTS flush_torrents")
		conn.execBuffer(&query)

		query.Reset()
		query.WriteString("CREATE TEMPORARY TABLE flush_torrents (" +
			"ID int(10) NOT NULL, " +
			"Snatched int(10) unsigned NOT NULL DEFAULT 0, " +
			"Seeders int(6) NOT NULL DEFAULT 0, " +
			"Leechers int(6) NOT NULL DEFAULT 0, " +
			"last_action int(11) NOT NULL DEFAULT 0)")
		conn.execBuffer(&query)

		query.Reset()
		query.WriteString("INSERT INTO flush_torrents VALUES\n")

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

			conn.execBuffer(&query)

			query.Reset()
			query.WriteString("UPDATE torrents t, flush_torrents ft SET " +
				"t.Snatched = t.Snatched + ft.Snatched, " +
				"t.Seeders = ft.Seeders, " +
				"t.Leechers = ft.Leechers, " +
				"t.last_action = IF(t.last_action < ft.last_action, ft.last_action, t.last_action)" +
				"WHERE t.ID = ft.ID")
			conn.execBuffer(&query)

			elapsedTime := time.Since(startTime)
			collectors.UpdateFlushTime("torrents", elapsedTime)
			collectors.UpdateChannelsLen("torrents", count)

			if length < (config.TorrentFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
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

	conn := OpenDatabaseConnection()

	for {
		length := util.Max(1, len(db.userChannel))

		query.Reset()
		query.WriteString("DROP TABLE IF EXISTS flush_users")
		conn.execBuffer(&query)

		query.Reset()
		query.WriteString("CREATE TEMPORARY TABLE flush_users (" +
			"ID int(10) unsigned NOT NULL, " +
			"Uploaded bigint(20) unsigned NOT NULL DEFAULT 0, " +
			"Downloaded bigint(20) unsigned NOT NULL DEFAULT 0, " +
			"rawdl bigint(20) NOT NULL, " +
			"rawup bigint(20) NOT NULL)")
		conn.execBuffer(&query)

		query.Reset()
		query.WriteString("INSERT INTO flush_users VALUES\n")

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

			conn.execBuffer(&query)

			query.Reset()
			query.WriteString("UPDATE users_main u, flush_users fu SET " +
				"u.Uploaded = u.Uploaded + fu.Uploaded, " +
				"u.Downloaded = u.Downloaded + fu.Downloaded, " +
				"u.rawdl = u.rawdl + fu.rawdl, " +
				"u.rawup = u.rawup + fu.rawup " +
				"WHERE u.ID = fu.ID")
			conn.execBuffer(&query)

			elapsedTime := time.Since(startTime)
			collectors.UpdateFlushTime("users", elapsedTime)
			collectors.UpdateChannelsLen("users", count)

			if length < (config.UserFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
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

	conn := OpenDatabaseConnection()

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

			conn.execBuffer(&query)
			db.transferHistoryWaitGroup.Done()

			elapsedTime := time.Since(startTime)
			collectors.UpdateFlushTime("transfer_history", elapsedTime)
			collectors.UpdateChannelsLen("transfer_history", count)

			if length < (config.TransferHistoryFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
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

	conn := OpenDatabaseConnection()

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
			conn.execBuffer(&query)

			elapsedTime := time.Since(startTime)
			collectors.UpdateFlushTime("transfer_ips", elapsedTime)
			collectors.UpdateChannelsLen("transfer_ips", count)

			if length < (config.TransferIpsFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
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

	conn := OpenDatabaseConnection()

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
			conn.execBuffer(&query)

			elapsedTime := time.Since(startTime)
			collectors.UpdateFlushTime("snatches", elapsedTime)
			collectors.UpdateChannelsLen("snatches", count)

			if length < (config.SnatchFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
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

		oldestActive := now - 2*int64(config.AnnounceInterval.Seconds())

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
		log.Info.Printf("Purged %d inactive peers from memory (%dms)\n", count, elapsedTime.Nanoseconds()/1000000)

		// Wait on flushing to prevent a race condition where the user has announced but their announce time hasn't been flushed yet
		db.goTransferHistoryWait()

		// Then set them to inactive in the database
		db.mainConn.mutex.Lock()

		start = time.Now()
		result := db.mainConn.exec(db.cleanStalePeersStmt, oldestActive)

		rows, err := result.RowsAffected()
		if err != nil {
			log.Error.Printf("Error in getting affected rows: %s", err)
			log.WriteStack()
		}
		db.mainConn.mutex.Unlock()

		log.Info.Printf("Updated %d inactive peers in database (%dms)\n", rows, time.Since(start).Nanoseconds()/1000000)

		db.waitGroup.Done()
		time.Sleep(config.PurgeInactiveInterval)
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

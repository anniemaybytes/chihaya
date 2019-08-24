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
	"chihaya/config"
	"chihaya/util"
	"log"
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
 * TODO: investigate good wait timings
 */

func (db *Database) startFlushing() {
	db.torrentChannel = make(chan *bytes.Buffer, config.TorrentFlushBufferSize)
	db.userChannel = make(chan *bytes.Buffer, config.UserFlushBufferSize)
	db.transferHistoryChannel = make(chan *bytes.Buffer, config.TransferHistoryFlushBufferSize)
	db.transferIpsChannel = make(chan *bytes.Buffer, config.TransferIpsFlushBufferSize)
	db.snatchChannel = make(chan *bytes.Buffer, config.SnatchFlushBufferSize)

	go db.flushTorrents()
	go db.flushUsers()
	go db.flushTransferHistory()
	go db.flushTransferIps()
	go db.flushSnatches()

	go db.purgeInactivePeers()
}

func (db *Database) flushTorrents() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		length := util.Max(1, len(db.torrentChannel))
		query.Reset()

		query.WriteString("INSERT INTO torrents (ID, Snatched, Seeders, Leechers, last_action) VALUES\n")

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

		if config.LogFlushes && !db.terminate {
			log.Printf("[torrents] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE Snatched = Snatched + VALUE(Snatched), " +
				"Seeders = VALUE(Seeders), Leechers = VALUE(Leechers), " +
				"last_action = IF(last_action < VALUE(last_action), VALUE(last_action), last_action);")

			conn.execBuffer(&query)

			if length < (config.TorrentFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushUsers() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		length := util.Max(1, len(db.userChannel))
		query.Reset()

		query.WriteString("INSERT INTO users_main (ID, Uploaded, Downloaded, rawdl, rawup) VALUES\n")

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

		if config.LogFlushes && !db.terminate {
			log.Printf("[users_main] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE Uploaded = Uploaded + VALUE(Uploaded), " +
				"Downloaded = Downloaded + VALUE(Downloaded), rawdl = rawdl + VALUE(rawdl), rawup = rawup + VALUE(rawup);")

			conn.execBuffer(&query)

			if length < (config.UserFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushTransferHistory() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		db.transferHistoryWaitGroupMu.Lock()
		if db.transferHistoryWaitGroupSe == 1 {
			db.transferHistoryWaitGroupMu.Unlock()
			log.Printf("goTransferHistoryWait has started... (skipping flushTransferHistory)")
			time.Sleep(time.Second)
			continue
		}
		db.transferHistoryWaitGroup.Add(1)
		db.transferHistoryWaitGroupMu.Unlock()

		length := util.Max(1, len(db.transferHistoryChannel))
		query.Reset()

		query.WriteString("INSERT INTO transfer_history (uid, fid, uploaded, downloaded, " +
			"seeding, starttime, last_announce, activetime, seedtime, active, snatched, remaining) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.transferHistoryChannel
			if b == nil {
				break
			}
			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if config.LogFlushes && !db.terminate {
			log.Printf("[transfer_history] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE uploaded = uploaded + VALUE(uploaded), " +
				"downloaded = downloaded + VALUE(downloaded), connectable = VALUE(connectable), " +
				"seeding = VALUE(seeding), activetime = activetime + VALUE(activetime), " +
				"seedtime = seedtime + VALUE(seedtime), last_announce = VALUE(last_announce), " +
				"active = VALUE(active), snatched = snatched + VALUE(snatched), remaining = VALUE(remaining);")

			conn.execBuffer(&query)
			db.transferHistoryWaitGroup.Done()

			if length < (config.TransferHistoryFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			db.transferHistoryWaitGroup.Done()
			break
		} else {
			db.transferHistoryWaitGroup.Done()
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushTransferIps() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		length := util.Max(1, len(db.transferIpsChannel))
		query.Reset()

		query.WriteString("INSERT INTO transfer_ips (uid, fid, client_id, ip, uploaded, downloaded, starttime, last_announce) VALUES\n")

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

		if config.LogFlushes && !db.terminate {
			log.Printf("[transfer_ips] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE downloaded = downloaded + VALUE(downloaded), uploaded = uploaded + VALUE(uploaded), last_announce = VALUE(last_announce);")
			conn.execBuffer(&query)

			if length < (config.TransferIpsFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushSnatches() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
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

		if config.LogFlushes && !db.terminate {
			log.Printf("[snatches] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE snatched_time = VALUE(snatched_time);")

			conn.execBuffer(&query)

			if length < (config.SnatchFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
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

		log.Printf("Purged %d inactive peers from memory (%dms)\n", count, time.Now().Sub(start).Nanoseconds()/1000000)

		// Wait on flushing to prevent a race condition where the user has announced but their announce time hasn't been flushed yet
		db.goTransferHistoryWait()

		// Then set them to inactive in the database
		db.mainConn.mutex.Lock()
		start = time.Now()
		result := db.mainConn.exec(db.cleanStalePeersStmt, oldestActive)
		rows := result.AffectedRows()
		db.mainConn.mutex.Unlock()

		log.Printf("Updated %d inactive peers in database (%dms)\n", rows, time.Now().Sub(start).Nanoseconds()/1000000)

		db.waitGroup.Done()
		time.Sleep(config.PurgeInactiveInterval)
	}
}

func (db *Database) goTransferHistoryWait() {
	log.Printf("Starting goTransferHistoryWait")
	db.transferHistoryWaitGroupMu.Lock()
	db.transferHistoryWaitGroupSe = 1
	db.transferHistoryWaitGroupMu.Unlock()
	db.transferHistoryWaitGroup.Wait()
	db.transferHistoryWaitGroupMu.Lock()
	db.transferHistoryWaitGroupSe = 0
	db.transferHistoryWaitGroupMu.Unlock()
	log.Printf("Releasing goTransferHistoryWait")
}

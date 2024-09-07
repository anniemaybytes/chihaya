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
	"errors"
	"log/slog"
	"time"

	"chihaya/collector"
	"chihaya/config"
	cdb "chihaya/database/types"
	"chihaya/util"
)

var (
	peerInactivityInterval     int
	purgeInactivePeersInterval int
	flushSleepInterval         int
	logFlushes                 bool
)

func init() {
	intervals := config.Section("intervals")

	peerInactivityInterval, _ = intervals.GetInt("peer_inactivity", 3900)
	purgeInactivePeersInterval, _ = intervals.GetInt("purge_inactive_peers", 120)
	flushSleepInterval, _ = intervals.GetInt("flush", 5)

	logFlushes, _ = config.GetBool("log_flushes", true)
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

	errDbTerminate       = errors.New("shutting down database connection")
	errGotNilFromChannel = errors.New("got nil while receiving from non-empty channel")
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

	go func() {
		time.Sleep(2 * time.Second)

		db.purgeInactivePeers()
	}()
}

func (db *Database) closeFlushChannels() {
	close(db.torrentChannel)
	close(db.userChannel)
	close(db.transferHistoryChannel)
	close(db.transferIpsChannel)
	close(db.snatchChannel)
}

func (db *Database) flushTorrents() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	for {
		query.Reset()
		query.WriteString("INSERT IGNORE INTO torrents (ID, Snatched, Seeders, Leechers, last_action) VALUES ")

		length := len(db.torrentChannel)

		for count = 0; count < length; count++ {
			b := <-db.torrentChannel
			if b == nil {
				panic(errGotNilFromChannel)
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if count > 0 {
			if logFlushes && !db.terminate.Load() {
				slog.Info("flushing", "channel", "torrents", "count", count)
			}

			startTime := time.Now()

			query.WriteString(" ON DUPLICATE KEY UPDATE Snatched = Snatched + VALUE(Snatched), " +
				"Seeders = VALUE(Seeders), Leechers = VALUE(Leechers), " +
				"last_action = IF(last_action < VALUE(last_action), VALUE(last_action), last_action)")
			db.exec(&query)

			if !db.terminate.Load() {
				collector.UpdateChannelFlushTime("torrents", time.Since(startTime))
				collector.UpdateChannelFlushLen("torrents", count)
			}

			if length < (torrentFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate.Load() {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushUsers() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	for {
		query.Reset()
		query.WriteString("INSERT IGNORE INTO users_main (ID, Uploaded, Downloaded, rawdl, rawup) VALUES ")

		length := len(db.userChannel)

		for count = 0; count < length; count++ {
			b := <-db.userChannel
			if b == nil {
				panic(errGotNilFromChannel)
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if count > 0 {
			if logFlushes && !db.terminate.Load() {
				slog.Info("flushing", "channel", "users", "count", count)
			}

			startTime := time.Now()

			query.WriteString(" ON DUPLICATE KEY UPDATE Uploaded = Uploaded + VALUE(Uploaded), " +
				"Downloaded = Downloaded + VALUE(Downloaded), rawdl = rawdl + VALUE(rawdl), rawup = rawup + VALUE(rawup)")
			db.exec(&query)

			if !db.terminate.Load() {
				collector.UpdateChannelFlushTime("users", time.Since(startTime))
				collector.UpdateChannelFlushLen("users", count)
			}

			if length < (userFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate.Load() {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushTransferHistory() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	for {
		length, err := func() (int, error) {
			db.transferHistoryLock.Lock()
			defer db.transferHistoryLock.Unlock()

			query.Reset()
			query.WriteString("INSERT INTO transfer_history (uid, fid, uploaded, downloaded, " +
				"seeding, starttime, last_announce, activetime, seedtime, active, snatched, remaining) VALUES\n")

			length := len(db.transferHistoryChannel)

			for count = 0; count < length; count++ {
				b := <-db.transferHistoryChannel
				if b == nil {
					panic(errGotNilFromChannel)
				}

				query.Write(b.Bytes())
				db.bufferPool.Give(b)

				if count != length-1 {
					query.WriteRune(',')
				}
			}

			if count > 0 {
				if logFlushes && !db.terminate.Load() {
					slog.Info("flushing", "channel", "transfer_history", "count", count)
				}

				startTime := time.Now()

				query.WriteString("\nON DUPLICATE KEY UPDATE uploaded = uploaded + VALUE(uploaded), " +
					"downloaded = downloaded + VALUE(downloaded), remaining = VALUE(remaining), " +
					"seeding = VALUE(seeding), activetime = activetime + VALUE(activetime), " +
					"seedtime = seedtime + VALUE(seedtime), last_announce = VALUE(last_announce), " +
					"active = VALUE(active), snatched = snatched + VALUE(snatched);")

				db.exec(&query)

				if !db.terminate.Load() {
					collector.UpdateChannelFlushTime("transfer_history", time.Since(startTime))
					collector.UpdateChannelFlushLen("transfer_history", count)
				}

				return length, nil
			} else if db.terminate.Load() {
				return 0, errDbTerminate
			}

			return length, nil
		}()

		if err != nil {
			break
		} else if length < (transferHistoryFlushBufferSize >> 1) {
			time.Sleep(time.Duration(flushSleepInterval) * time.Second)
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushTransferIps() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	for {
		query.Reset()
		query.WriteString("INSERT INTO transfer_ips (uid, fid, client_id, ip, port, uploaded, downloaded, " +
			"starttime, last_announce) VALUES\n")

		length := len(db.transferIpsChannel)

		for count = 0; count < length; count++ {
			b := <-db.transferIpsChannel
			if b == nil {
				panic(errGotNilFromChannel)
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if count > 0 {
			if logFlushes && !db.terminate.Load() {
				slog.Info("flushing", "channel", "transfer_ips", "count", count)
			}

			startTime := time.Now()

			// todo: port should be part of PK
			query.WriteString("\nON DUPLICATE KEY UPDATE port = VALUE(port), downloaded = downloaded + VALUE(downloaded), " +
				"uploaded = uploaded + VALUE(uploaded), last_announce = VALUE(last_announce)")
			db.exec(&query)

			if !db.terminate.Load() {
				collector.UpdateChannelFlushTime("transfer_ips", time.Since(startTime))
				collector.UpdateChannelFlushLen("transfer_ips", count)
			}

			if length < (transferIpsFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate.Load() {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushSnatches() {
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()

	var (
		query bytes.Buffer
		count int
	)

	for {
		query.Reset()
		query.WriteString("INSERT INTO transfer_history (uid, fid, snatched_time) VALUES\n")

		length := len(db.snatchChannel)

		for count = 0; count < length; count++ {
			b := <-db.snatchChannel
			if b == nil {
				panic(errGotNilFromChannel)
			}

			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if count > 0 {
			if logFlushes && !db.terminate.Load() {
				slog.Info("flushing", "channel", "snatches", "count", count)
			}

			startTime := time.Now()

			query.WriteString("\nON DUPLICATE KEY UPDATE snatched_time = VALUE(snatched_time)")
			db.exec(&query)

			if !db.terminate.Load() {
				collector.UpdateChannelFlushTime("snatches", time.Since(startTime))
				collector.UpdateChannelFlushLen("snatches", count)
			}

			if length < (snatchFlushBufferSize >> 1) {
				time.Sleep(time.Duration(flushSleepInterval) * time.Second)
			}
		} else if db.terminate.Load() {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) purgeInactivePeers() {
	var (
		startTime time.Time
		count     int
	)

	util.ContextTick(db.ctx, time.Duration(purgeInactivePeersInterval)*time.Second, func() {
		startTime = time.Now()
		count = 0

		oldestActive := time.Now().Unix() - int64(peerInactivityInterval)

		// First, remove inactive peers from memory
		dbTorrents := *db.Torrents.Load()
		for _, torrent := range dbTorrents {
			func() {
				torrent.PeerLock() // Take write lock to operate on peer entries
				defer torrent.PeerUnlock()

				countThisTorrent := count

				for id, peer := range torrent.Leechers {
					if peer.LastAnnounce < oldestActive {
						delete(torrent.Leechers, id)

						count++
					}
				}

				if countThisTorrent != count && len(torrent.Leechers) == 0 {
					/* Deallocate previous map since Go does not free space used on maps when deleting objects.
					We're doing it only for Leechers as potential advantage from freeing one or two Seeders is
					virtually nil, while Leechers can incur significant memory leaks due to initial swarm activity. */
					torrent.Leechers = make(map[cdb.PeerKey]*cdb.Peer)
				}

				for id, peer := range torrent.Seeders {
					if peer.LastAnnounce < oldestActive {
						delete(torrent.Seeders, id)

						count++
					}
				}

				if countThisTorrent != count {
					// Update lengths of peers
					torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
					torrent.LeechersLength.Store(uint32(len(torrent.Leechers)))

					db.QueueTorrent(torrent, 0)
				}
			}()
		}

		elapsedTime := time.Since(startTime)
		collector.UpdatePurgeInactivePeersTime(elapsedTime)
		slog.Info("purged inactive peers from memory", "count", count, "elapsed", elapsedTime)

		// Set peers as inactive in the database
		func() {
			db.waitGroup.Add(1)
			defer db.waitGroup.Done()

			// Wait to prevent a race condition where the user has announced but their announce time hasn't been flushed yet
			db.transferHistoryLock.Lock()
			defer db.transferHistoryLock.Unlock()

			startTime = time.Now()

			result := db.execute(db.cleanStalePeersStmt, oldestActive)
			if result != nil {
				rows, _ := result.RowsAffected()
				slog.Info("updated inactive peers in database", "rows", rows, "elapsed", time.Since(startTime))
			}
		}()
	})
}

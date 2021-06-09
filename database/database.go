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
	"chihaya/database/types"
	"chihaya/log"
	"chihaya/util"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Connection struct {
	sqlDb *sql.DB
	mutex sync.Mutex
}

type Database struct {
	TorrentsMutex sync.RWMutex

	snatchChannel          chan *bytes.Buffer
	transferHistoryChannel chan *bytes.Buffer
	transferIpsChannel     chan *bytes.Buffer

	loadTorrentsStmt    *sql.Stmt
	loadClientsStmt     *sql.Stmt
	loadFreeleechStmt   *sql.Stmt
	cleanStalePeersStmt *sql.Stmt
	unPruneTorrentStmt  *sql.Stmt

	Users map[string]*types.User

	loadHnrStmt *sql.Stmt

	HitAndRuns map[types.UserTorrentPair]struct{}
	Torrents   map[string]*types.Torrent // SHA-1 hash (20 bytes)

	loadUsersStmt *sql.Stmt

	Clients map[uint16]string

	mainConn *Connection // Used for reloading and misc queries

	torrentChannel chan *bytes.Buffer
	userChannel    chan *bytes.Buffer

	bufferPool *util.BufferPool

	ClientsMutex sync.RWMutex
	UsersMutex   sync.RWMutex

	waitGroup sync.WaitGroup

	transferHistoryWaitGroup   sync.WaitGroup
	transferHistoryWaitGroupMu sync.Mutex
	transferHistoryWaitGroupSe uint8

	terminate bool
}

var (
	deadlockWaitTime   int
	maxDeadlockRetries int
)

var defaultDsn = map[string]string{
	"username": "chihaya",
	"password": "",
	"proto":    "tcp",
	"addr":     "127.0.0.1:3306",
	"database": "chihaya",
}

func (db *Database) Init() {
	db.terminate = false

	log.Info.Printf("Opening database connection...")

	db.mainConn = Open()

	maxBuffers := torrentFlushBufferSize +
		userFlushBufferSize +
		transferHistoryFlushBufferSize +
		transferIpsFlushBufferSize +
		snatchFlushBufferSize

	// Used for recording updates, so the max required size should be < 128 bytes. See record.go for details
	db.bufferPool = util.NewBufferPool(maxBuffers, 128)

	var err error

	db.loadUsersStmt, err = db.mainConn.sqlDb.Prepare(
		"SELECT ID, torrent_pass, DownMultiplier, UpMultiplier, DisableDownload, TrackerHide " +
			"FROM users_main " +
			"WHERE Enabled = '1'")
	if err != nil {
		panic(err)
	}

	db.loadHnrStmt, err = db.mainConn.sqlDb.Prepare(
		"SELECT h.uid, h.fid FROM transfer_history AS h " +
			"JOIN users_main AS u ON u.ID = h.uid " +
			"WHERE h.hnr = 1 AND u.Enabled = '1'")
	if err != nil {
		panic(err)
	}

	db.loadTorrentsStmt, err = db.mainConn.sqlDb.Prepare(
		"SELECT t.ID, t.info_hash, (IFNULL(tg.DownMultiplier, 1) * t.DownMultiplier), " +
			"(IFNULL(tg.UpMultiplier, 1) * t.UpMultiplier), t.Snatched, t.Status " +
			"FROM torrents AS t " +
			"LEFT JOIN torrent_group_freeleech AS tg ON tg.GroupID = t.GroupID AND tg.Type = t.TorrentType")
	if err != nil {
		panic(err)
	}

	db.loadClientsStmt, err = db.mainConn.sqlDb.Prepare(
		"SELECT id, peer_id FROM approved_clients WHERE archived = 0")
	if err != nil {
		panic(err)
	}

	db.loadFreeleechStmt, err = db.mainConn.sqlDb.Prepare(
		"SELECT mod_setting FROM mod_core WHERE mod_option = 'global_freeleech'")
	if err != nil {
		panic(err)
	}

	db.cleanStalePeersStmt, err = db.mainConn.sqlDb.Prepare(
		"UPDATE transfer_history SET active = 0 WHERE last_announce < ? AND active = 1")
	if err != nil {
		panic(err)
	}

	db.unPruneTorrentStmt, err = db.mainConn.sqlDb.Prepare(
		"UPDATE torrents SET Status = 0 WHERE ID = ?")
	if err != nil {
		panic(err)
	}

	db.Users = make(map[string]*types.User)
	db.HitAndRuns = make(map[types.UserTorrentPair]struct{})
	db.Torrents = make(map[string]*types.Torrent)
	db.Clients = make(map[uint16]string)

	db.deserialize()

	// Run initial load to populate data in memory before we start accepting connections
	log.Info.Printf("Populating initial data into memory, please wait...")
	db.loadUsers()
	db.loadHitAndRuns()
	db.loadTorrents()
	db.loadConfig()
	db.loadClients()

	log.Info.Printf("Starting goroutines...")
	db.startReloading()
	db.startSerializing()
	db.startFlushing()
}

func (db *Database) Terminate() {
	db.terminate = true

	close(db.torrentChannel)
	close(db.userChannel)
	close(db.transferHistoryChannel)
	close(db.transferIpsChannel)
	close(db.snatchChannel)

	go func() {
		time.Sleep(10 * time.Second)
		log.Info.Printf("Waiting for database flushing to finish. This can take a few minutes, please be patient!")
	}()

	db.waitGroup.Wait()
	db.mainConn.mutex.Lock()
	_ = db.mainConn.Close()
	db.mainConn.mutex.Unlock()
	db.serialize()
}

func Open() *Connection {
	databaseConfig := config.Section("database")
	deadlockWaitTime, _ = databaseConfig.GetInt("deadlock_pause", 1)
	maxDeadlockRetries, _ = databaseConfig.GetInt("deadlock_retries", 5)

	channelsConfig := config.Section("channels")
	torrentFlushBufferSize, _ = channelsConfig.GetInt("torrent", 5000)
	userFlushBufferSize, _ = channelsConfig.GetInt("user", 5000)
	transferHistoryFlushBufferSize, _ = channelsConfig.GetInt("transfer_history", 5000)
	transferIpsFlushBufferSize, _ = channelsConfig.GetInt("transfer_ips", 5000)
	snatchFlushBufferSize, _ = channelsConfig.GetInt("snatch", 25)

	// DSN Format: username:password@protocol(address)/dbname?param=value
	// First try to load the DSN from environment. USeful for tests.
	databaseDsn := os.Getenv("DB_DSN")
	if databaseDsn == "" {
		dbUsername, _ := databaseConfig.Get("username", defaultDsn["username"])
		dbPassword, _ := databaseConfig.Get("password", defaultDsn["password"])
		dbProto, _ := databaseConfig.Get("proto", defaultDsn["proto"])
		dbAddr, _ := databaseConfig.Get("addr", defaultDsn["addr"])
		dbDatabase, _ := databaseConfig.Get("database", defaultDsn["database"])
		databaseDsn = fmt.Sprintf("%s:%s@%s(%s)/%s",
			dbUsername,
			dbPassword,
			dbProto,
			dbAddr,
			dbDatabase,
		)
	}

	sqlDb, err := sql.Open("mysql", databaseDsn)
	if err != nil {
		log.Fatal.Fatalf("Couldn't connect to database - %s", err)
	}

	err = sqlDb.Ping()
	if err != nil {
		log.Fatal.Fatalf("Couldn't ping database - %s", err)
	}

	return &Connection{
		sqlDb: sqlDb,
	}
}

func (db *Connection) Close() error {
	return db.sqlDb.Close()
}

func (db *Connection) query(stmt *sql.Stmt, args ...interface{}) *sql.Rows { //nolint:unparam
	rows, _ := perform(func() (interface{}, error) {
		return stmt.Query(args...)
	}).(*sql.Rows)

	return rows
}

func (db *Connection) execute(stmt *sql.Stmt, args ...interface{}) sql.Result {
	result, _ := perform(func() (interface{}, error) {
		return stmt.Exec(args...)
	}).(sql.Result)

	return result
}

func (db *Connection) exec(query *bytes.Buffer, args ...interface{}) sql.Result { //nolint:unparam
	result, _ := perform(func() (interface{}, error) {
		return db.sqlDb.Exec(query.String(), args...)
	}).(sql.Result)

	return result
}

func perform(exec func() (interface{}, error)) (result interface{}) {
	var (
		err   error
		tries int
		wait  time.Duration
	)

	for tries = 1; tries <= maxDeadlockRetries; tries++ {
		result, err = exec()
		if err != nil {
			if merr, isMysqlError := err.(*mysql.MySQLError); isMysqlError {
				if merr.Number == 1213 || merr.Number == 1205 {
					wait = time.Duration(deadlockWaitTime*tries) * time.Second
					log.Warning.Printf("Deadlock found! Retrying in %s (%d/%d)", wait.String(), tries,
						maxDeadlockRetries)

					if tries == 1 {
						collectors.IncrementDeadlockCount()
					}

					collectors.IncrementDeadlockTime(wait)
					time.Sleep(wait)

					continue
				} else {
					log.Error.Printf("SQL error %d: %s", merr.Number, merr.Message)
					log.WriteStack()

					collectors.IncrementSQLErrorCount()
				}
			} else {
				log.Panic.Printf("Error executing SQL: %s", err)
				panic(err)
			}
		}

		return
	}

	log.Error.Printf("Deadlocked %d times, giving up!", tries)
	log.WriteStack()
	collectors.IncrementDeadlockAborted()

	return
}

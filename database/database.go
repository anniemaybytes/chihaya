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
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Peer struct {
	Seeding      bool
	ClientId     uint16
	Port         uint16
	UserId       uint32
	Ip           uint32
	TorrentId    uint64
	Uploaded     uint64
	Downloaded   uint64
	Left         uint64
	StartTime    int64 // unix time
	LastAnnounce int64
	Id           string
	IpAddr       string
	Addr         []byte
}

type Torrent struct {
	Status         uint8
	Snatched       uint16
	Id             uint64
	LastAction     int64
	UpMultiplier   float64
	DownMultiplier float64

	Seeders  map[string]*Peer
	Leechers map[string]*Peer
}

type User struct {
	DisableDownload bool
	TrackerHide     bool
	Id              uint32
	UpMultiplier    float64
	DownMultiplier  float64
}

type UserTorrentPair struct {
	UserId    uint32
	TorrentId uint64
}

type DatabaseConnection struct {
	sqlDb *sql.DB
	mutex sync.Mutex
}

type Database struct {
	TorrentsMutex sync.RWMutex

	snatchChannel          chan *bytes.Buffer
	transferHistoryChannel chan *bytes.Buffer
	transferIpsChannel     chan *bytes.Buffer

	loadTorrentsStmt    *sql.Stmt
	loadWhitelistStmt   *sql.Stmt
	loadFreeleechStmt   *sql.Stmt
	cleanStalePeersStmt *sql.Stmt
	unPruneTorrentStmt  *sql.Stmt

	Users map[string]*User // 32 bytes

	loadHnrStmt *sql.Stmt

	HitAndRuns map[UserTorrentPair]struct{}
	Torrents   map[string]*Torrent // SHA-1 hash (20 bytes)

	loadUsersStmt *sql.Stmt

	Whitelist map[uint16]string

	mainConn *DatabaseConnection // Used for reloading and misc queries

	torrentChannel chan *bytes.Buffer
	userChannel    chan *bytes.Buffer

	bufferPool *util.BufferPool

	WhitelistMutex sync.RWMutex
	UsersMutex     sync.RWMutex

	waitGroup sync.WaitGroup

	transferHistoryWaitGroup   sync.WaitGroup
	transferHistoryWaitGroupMu sync.Mutex
	transferHistoryWaitGroupSe uint8

	terminate bool
}

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

	db.mainConn = OpenDatabaseConnection()

	maxBuffers := config.TorrentFlushBufferSize + config.UserFlushBufferSize + config.TransferHistoryFlushBufferSize +
		config.TransferIpsFlushBufferSize + config.SnatchFlushBufferSize

	// Used for recording updates, so the max required size should be < 128 bytes. See record.go for details
	db.bufferPool = util.NewBufferPool(maxBuffers, 128)

	db.loadUsersStmt = db.mainConn.prepareStatement("SELECT ID, torrent_pass, DownMultiplier, UpMultiplier, DisableDownload, TrackerHide FROM users_main WHERE Enabled='1'")
	db.loadHnrStmt = db.mainConn.prepareStatement("SELECT h.uid,h.fid FROM transfer_history AS h JOIN users_main AS u ON u.ID = h.uid WHERE h.hnr='1' AND u.Enabled='1'")
	db.loadTorrentsStmt = db.mainConn.prepareStatement("SELECT t.ID ID, t.info_hash info_hash, (IFNULL(tg.DownMultiplier,1) * t.DownMultiplier) DownMultiplier, (IFNULL(tg.UpMultiplier,1) * t.UpMultiplier) UpMultiplier, t.Snatched Snatched, t.Status Status FROM torrents AS t LEFT JOIN torrent_group_freeleech AS tg ON tg.GroupID=t.GroupID AND tg.Type=t.TorrentType")
	db.loadWhitelistStmt = db.mainConn.prepareStatement("SELECT id, peer_id FROM client_whitelist WHERE archived = 0")
	db.loadFreeleechStmt = db.mainConn.prepareStatement("SELECT mod_setting FROM mod_core WHERE mod_option='global_freeleech'")
	db.cleanStalePeersStmt = db.mainConn.prepareStatement("UPDATE transfer_history SET active = '0' WHERE last_announce < ? AND active='1'")
	db.unPruneTorrentStmt = db.mainConn.prepareStatement("UPDATE torrents SET Status=0 WHERE ID = ?")

	db.Users = make(map[string]*User)
	db.HitAndRuns = make(map[UserTorrentPair]struct{})
	db.Torrents = make(map[string]*Torrent)
	db.Whitelist = make(map[uint16]string)

	db.deserialize()

	// Run initial load to populate data in memory before we start accepting connections
	log.Info.Printf("Populating initial data into memory, please wait...")
	db.loadUsers()
	db.loadHitAndRuns()
	db.loadTorrents()
	db.loadConfig()
	db.loadWhitelist()

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

func OpenDatabaseConnection() (db *DatabaseConnection) {
	db = &DatabaseConnection{}
	databaseConfig := config.Section("database")
	databaseDsn := fmt.Sprintf("%s:%s@%s(%s)/%s",
		databaseConfig.Get("username", defaultDsn["username"]),
		databaseConfig.Get("password", defaultDsn["password"]),
		databaseConfig.Get("proto", defaultDsn["proto"]),
		databaseConfig.Get("addr", defaultDsn["addr"]),
		databaseConfig.Get("database", defaultDsn["database"]),
	) // DSN Format: username:password@protocol(address)/dbname?param=value

	sqlDb, err := sql.Open("mysql", databaseDsn)
	if err != nil {
		log.Fatal.Fatalf("Couldn't connect to database at %s - %s", databaseDsn, err)
	}

	err = sqlDb.Ping()
	if err != nil {
		log.Fatal.Fatalf("Couldn't ping database at %s - %s", databaseDsn, err)
	}

	db.sqlDb = sqlDb

	return
}

func (db *DatabaseConnection) Close() error {
	return db.sqlDb.Close()
}

func (db *DatabaseConnection) prepareStatement(sql string) *sql.Stmt {
	stmt, err := db.sqlDb.Prepare(sql)
	if err != nil {
		log.Panic.Printf("%s for SQL: %s", err, sql)
		panic(err)
	}

	return stmt
}

func (db *DatabaseConnection) query(stmt *sql.Stmt, args ...interface{}) *sql.Rows {
	rows, _ := handleDeadlock(func() (interface{}, error) {
		return stmt.Query(args...)
	}).(*sql.Rows)

	return rows
}

func (db *DatabaseConnection) exec(stmt *sql.Stmt, args ...interface{}) sql.Result {
	result, _ := handleDeadlock(func() (interface{}, error) {
		return stmt.Exec(args...)
	}).(sql.Result)

	return result
}

func (db *DatabaseConnection) execBuffer(query *bytes.Buffer, args ...interface{}) sql.Result {
	result, _ := handleDeadlock(func() (interface{}, error) {
		return db.sqlDb.Exec(query.String(), args...)
	}).(sql.Result)

	return result
}

func handleDeadlock(execFunc func() (interface{}, error)) (result interface{}) {
	var err error

	var tries int

	var wait int64

	for tries = 0; tries < config.MaxDeadlockRetries; tries++ {
		result, err = execFunc()
		if err != nil {
			if merr, isMysqlError := err.(*mysql.MySQLError); isMysqlError {
				if merr.Number == 1213 || merr.Number == 1205 {
					wait = config.DeadlockWaitTime.Nanoseconds() * int64(tries+1)
					log.Warning.Printf("Deadlock found! Retrying in %dms (%d/20)", wait/1000000, tries)
					collectors.IncrementDeadlockCount()
					collectors.IncrementDeadlockTime(time.Duration(wait))
					time.Sleep(time.Duration(wait))

					continue
				} else {
					log.Error.Printf("SQL error (CODE %d): %s", merr.Number, merr.Message)
				}
			} else {
				log.Panic.Printf("Error executing SQL: %s", err)
				panic(err)
			}
		}

		return
	}

	log.Error.Printf("Deadlocked %d times, giving up!", tries)

	return
}

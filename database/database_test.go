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
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	cdb "chihaya/database/types"

	"github.com/go-testfixtures/testfixtures/v3"
)

var (
	db       *Database
	fixtures *testfixtures.Loader
)

func TestMain(m *testing.M) {
	var err error

	flushSleepInterval = 1
	db = &Database{}

	db.Init()

	fixtures, err = testfixtures.New(
		testfixtures.Database(db.mainConn.sqlDb),
		testfixtures.Dialect("mariadb"),
		testfixtures.Directory("fixtures"),
		testfixtures.DangerousSkipTestDatabaseCheck(),
	)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestLoadUsers(t *testing.T) {
	prepareTestDatabase()

	db.Users = make(map[string]*cdb.User)
	users := map[string]*cdb.User{
		"mUztWMpBYNCqzmge6vGeEUGSrctJbgpQ": {
			ID:              1,
			UpMultiplier:    1,
			DownMultiplier:  1,
			TrackerHide:     false,
			DisableDownload: false,
		},
		"tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw": {
			ID:              2,
			UpMultiplier:    0.5,
			DownMultiplier:  2,
			TrackerHide:     true,
			DisableDownload: true,
		},
	}

	// Test with fresh data
	db.loadUsers()

	if len(db.Users) != len(users) {
		t.Fatal(fixtureFailure("Did not load all users as expected from fixture file", len(users), len(db.Users)))
	}

	for passkey, user := range users {
		if !reflect.DeepEqual(user, db.Users[passkey]) {
			t.Fatal(fixtureFailure(
				fmt.Sprintf("Did not load user (%s) as expected from fixture file", passkey),
				user,
				db.Users[passkey]))
		}
	}

	// Now test load on top of existing data
	oldUsers := db.Users

	db.loadUsers()

	if !reflect.DeepEqual(oldUsers, db.Users) {
		t.Fatal(fixtureFailure("Did not reload users as expected from fixture file", oldUsers, db.Users))
	}
}

func TestLoadHitAndRuns(t *testing.T) {
	prepareTestDatabase()

	dbHitAndRuns := make(map[cdb.UserTorrentPair]struct{})
	db.HitAndRuns.Store(&dbHitAndRuns)

	db.loadHitAndRuns()

	dbHitAndRuns = *db.HitAndRuns.Load()

	hnr := cdb.UserTorrentPair{
		UserID:    2,
		TorrentID: 2,
	}
	_, hnrExists := dbHitAndRuns[hnr]

	if len(dbHitAndRuns) != 1 {
		t.Fatal(fixtureFailure("Did not load all hit and runs as expected from fixture file",
			1,
			len(dbHitAndRuns)))
	}

	if !hnrExists {
		t.Fatal(fixtureFailure("Did not load hit and run as expected from fixture file", dbHitAndRuns, hnr))
	}
}

func TestLoadTorrents(t *testing.T) {
	prepareTestDatabase()

	db.Torrents = make(map[cdb.TorrentHash]*cdb.Torrent)

	torrents := map[cdb.TorrentHash]*cdb.Torrent{
		{114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56}: {
			ID:             1,
			Status:         1,
			Snatched:       2,
			DownMultiplier: 1,
			UpMultiplier:   1,
			Seeders:        map[cdb.PeerKey]*cdb.Peer{},
			Leechers:       map[cdb.PeerKey]*cdb.Peer{},
			Group: cdb.TorrentGroup{
				GroupID:     1,
				TorrentType: "anime",
			},
		},
		{22, 168, 45, 221, 87, 225, 140, 177, 94, 34, 242, 225, 196, 234, 222, 46, 187, 131, 177, 155}: {
			ID:             2,
			Status:         0,
			Snatched:       0,
			DownMultiplier: 2,
			UpMultiplier:   0.5,
			Seeders:        map[cdb.PeerKey]*cdb.Peer{},
			Leechers:       map[cdb.PeerKey]*cdb.Peer{},
			Group: cdb.TorrentGroup{
				GroupID:     1,
				TorrentType: "music",
			},
		},
		{89, 252, 84, 49, 177, 28, 118, 28, 148, 205, 62, 185, 8, 37, 234, 110, 109, 200, 165, 241}: {
			ID:             3,
			Status:         0,
			Snatched:       0,
			DownMultiplier: 1,
			UpMultiplier:   1,
			Seeders:        map[cdb.PeerKey]*cdb.Peer{},
			Leechers:       map[cdb.PeerKey]*cdb.Peer{},
			Group: cdb.TorrentGroup{
				GroupID:     2,
				TorrentType: "anime",
			},
		},
	}

	// Test with fresh data
	db.loadTorrents()

	if len(db.Torrents) != len(torrents) {
		t.Fatal(fixtureFailure("Did not load all torrents as expected from fixture file",
			len(torrents),
			len(db.Torrents)))
	}

	for hash, torrent := range torrents {
		if !reflect.DeepEqual(torrent, db.Torrents[hash]) {
			t.Fatal(fixtureFailure(
				fmt.Sprintf("Did not load torrent (%s) as expected from fixture file", hash),
				torrent,
				db.Torrents[hash]))
		}
	}

	// Now test load on top of existing data
	oldTorrents := db.Torrents

	db.loadTorrents()

	if !reflect.DeepEqual(oldTorrents, db.Torrents) {
		t.Fatal(fixtureFailure("Did not reload torrents as expected from fixture file", oldTorrents, db.Torrents))
	}
}

func TestLoadGroupsFreeleech(t *testing.T) {
	prepareTestDatabase()

	dbMap := make(map[cdb.TorrentGroup]*cdb.TorrentGroupFreeleech)
	db.TorrentGroupFreeleech.Store(&dbMap)

	torrentGroupFreeleech := map[cdb.TorrentGroup]*cdb.TorrentGroupFreeleech{
		{
			GroupID:     2,
			TorrentType: "anime",
		}: {
			DownMultiplier: 0,
			UpMultiplier:   2,
		},
	}

	// Test with fresh data
	db.loadGroupsFreeleech()

	dbMap = *db.TorrentGroupFreeleech.Load()

	if len(dbMap) != len(torrentGroupFreeleech) {
		t.Fatal(fixtureFailure("Did not load all torrent group freeleech data as expected from fixture file",
			len(torrentGroupFreeleech),
			len(dbMap)))
	}

	for group, freeleech := range torrentGroupFreeleech {
		if !reflect.DeepEqual(freeleech, dbMap[group]) {
			t.Fatal(fixtureFailure(
				fmt.Sprintf("Did not load torrent group freeleech data (%v) as expected from fixture file", group),
				freeleech,
				dbMap[group]))
		}
	}

	// Now test load on top of existing data
	oldTorrentGroupFreeleech := *db.TorrentGroupFreeleech.Load()

	db.loadGroupsFreeleech()

	dbMap = *db.TorrentGroupFreeleech.Load()

	if !reflect.DeepEqual(oldTorrentGroupFreeleech, dbMap) {
		t.Fatal(fixtureFailure(
			"Did not reload torrent group freeleech data as expected from fixture file",
			oldTorrentGroupFreeleech,
			dbMap))
	}
}

func TestLoadConfig(t *testing.T) {
	prepareTestDatabase()

	GlobalFreeleech.Store(false)

	db.loadConfig()

	if GlobalFreeleech.Load() {
		t.Fatal(fixtureFailure("Did not load config as expected from fixture file",
			false,
			true))
	}
}

func TestLoadClients(t *testing.T) {
	prepareTestDatabase()

	dbClients := make(map[uint16]string)
	db.Clients.Store(&dbClients)

	expectedClients := map[uint16]string{
		1: "-TR2",
		3: "-DE13",
	}

	db.loadClients()

	dbClients = *db.Clients.Load()

	if len(dbClients) != 2 {
		t.Fatal(fixtureFailure("Did not load all clients as expected from fixture file", 2, dbClients))
	}

	if !reflect.DeepEqual(expectedClients, dbClients) {
		t.Fatal(fixtureFailure("Did not load clients as expected from fixture file", expectedClients, dbClients))
	}
}

func TestUnPrune(t *testing.T) {
	prepareTestDatabase()

	h := cdb.TorrentHash{114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56}
	torrent := *db.Torrents[h]
	torrent.Status = 0

	db.UnPrune(db.Torrents[h])

	db.loadTorrents()

	if !reflect.DeepEqual(&torrent, db.Torrents[h]) {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Torrent (%x) was not unpruned properly", h),
			&torrent,
			db.Torrents[h]))
	}
}

func TestRecordAndFlushUsers(t *testing.T) {
	prepareTestDatabase()

	testUser := db.Users["tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw"]

	var (
		initUpload      int64
		initDownload    int64
		initRawUpload   int64
		initRawDownload int64

		upload      int64
		download    int64
		rawUpload   int64
		rawDownload int64

		deltaUpload      int64
		deltaDownload    int64
		deltaRawUpload   int64
		deltaRawDownload int64
	)

	deltaRawDownload = 83472
	deltaRawUpload = 245
	deltaDownload = int64(float64(deltaRawDownload) * testUser.DownMultiplier)
	deltaUpload = int64(float64(deltaRawUpload) * testUser.UpMultiplier)

	row := db.mainConn.sqlDb.QueryRow("SELECT Uploaded, Downloaded, rawup, rawdl "+
		"FROM users_main WHERE ID = ?", testUser.ID)

	err := row.Scan(&initUpload, &initDownload, &initRawUpload, &initRawDownload)
	if err != nil {
		panic(err)
	}

	db.RecordUser(testUser, deltaRawUpload, deltaRawDownload, deltaUpload, deltaDownload)

	for len(db.userChannel) > 0 {
		time.Sleep(time.Second)
	}
	time.Sleep(200 * time.Millisecond)

	row = db.mainConn.sqlDb.QueryRow("SELECT Uploaded, Downloaded, rawup, rawdl "+
		"FROM users_main WHERE ID = ?", testUser.ID)

	err = row.Scan(&upload, &download, &rawUpload, &rawDownload)
	if err != nil {
		panic(err)
	}

	if initDownload+deltaDownload != download {
		t.Fatal(fixtureFailure(
			"Delta download incorrectly updated in the database for user tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw",
			deltaDownload,
			download-initDownload,
		))
	}

	if initUpload+deltaUpload != upload {
		t.Fatal(fixtureFailure(
			"Delta upload incorrectly updated in the database for user tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw",
			deltaUpload,
			upload-initUpload,
		))
	}

	if initRawDownload+deltaRawDownload != rawDownload {
		t.Fatal(fixtureFailure(
			"Delta raw download incorrectly updated in the database for user tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw",
			deltaRawDownload,
			rawDownload-initRawDownload,
		))
	}

	if initRawUpload+deltaRawUpload != rawUpload {
		t.Fatal(fixtureFailure(
			"Delta raw upload incorrectly updated in the database for user tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw",
			deltaRawUpload,
			rawUpload-initRawUpload,
		))
	}
}

func TestRecordAndFlushTransferHistory(t *testing.T) {
	prepareTestDatabase()

	testPeer := &cdb.Peer{
		UserID:       1,
		TorrentID:    1,
		Seeding:      true,
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
		Left:         65000,
	}

	var (
		initRawUpload   int64
		initRawDownload int64
		initActiveTime  int64
		initSeedTime    int64
		initSnatch      int64
		initActive      bool

		rawUpload   int64
		rawDownload int64
		activeTime  int64
		seedTime    int64
		snatch      int64
		active      bool

		deltaRawUpload   int64
		deltaRawDownload int64
		deltaActiveTime  int64
		deltaSeedTime    int64
		deltaSnatch      uint8
	)

	deltaSnatch = 45
	deltaRawDownload = 83472
	deltaRawUpload = 245
	deltaActiveTime = 267
	deltaSeedTime = 15

	row := db.mainConn.sqlDb.QueryRow("SELECT uploaded, downloaded, activetime, seedtime, active, snatched "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", testPeer.UserID, testPeer.TorrentID)

	err := row.Scan(&initRawUpload, &initRawDownload, &initActiveTime, &initSeedTime, &initActive, &initSnatch)
	if err != nil {
		panic(err)
	}

	db.RecordTransferHistory(testPeer,
		deltaRawUpload,
		deltaRawDownload,
		deltaActiveTime,
		deltaSeedTime,
		deltaSnatch,
		!initActive)

	for len(db.transferHistoryChannel) > 0 {
		time.Sleep(time.Second)
	}
	time.Sleep(200 * time.Millisecond)

	row = db.mainConn.sqlDb.QueryRow("SELECT uploaded, downloaded, activetime, seedtime, active, snatched "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", testPeer.UserID, testPeer.TorrentID)

	err = row.Scan(&rawUpload, &rawDownload, &activeTime, &seedTime, &active, &snatch)
	if err != nil {
		panic(err)
	}

	if initActive == active {
		t.Fatal(fixtureFailure("Active status incorrectly updated in the database", initActive, active))
	}

	if initSnatch+int64(deltaSnatch) != snatch {
		t.Fatal(fixtureFailure(
			"Delta snatches incorrectly updated in the database",
			deltaSnatch,
			snatch-initSnatch,
		))
	}

	if initActiveTime+deltaActiveTime != activeTime {
		t.Fatal(fixtureFailure(
			"Delta active time incorrectly updated in the database",
			deltaActiveTime,
			activeTime-initActiveTime,
		))
	}

	if initSeedTime+deltaSeedTime != seedTime {
		t.Fatal(fixtureFailure(
			"Delta seed time incorrectly updated in the database",
			deltaSeedTime,
			seedTime-initSeedTime,
		))
	}

	if initRawDownload+deltaRawDownload != rawDownload {
		t.Fatal(fixtureFailure(
			"Delta raw incorrectly updated in the database",
			deltaRawDownload,
			rawDownload-initRawDownload,
		))
	}

	if initRawUpload+deltaRawUpload != rawUpload {
		t.Fatal(fixtureFailure(
			"Delta raw upload incorrectly updated in the database",
			deltaRawUpload,
			rawUpload-initRawUpload,
		))
	}

	// Check if existing peer being updated properly
	gotPeer := &cdb.Peer{
		UserID:    testPeer.UserID,
		TorrentID: testPeer.TorrentID,
		StartTime: testPeer.StartTime,
	}

	var gotStartTime int64

	row = db.mainConn.sqlDb.QueryRow("SELECT seeding, starttime, last_announce, remaining "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", gotPeer.UserID, gotPeer.TorrentID)

	err = row.Scan(&gotPeer.Seeding, &gotStartTime, &gotPeer.LastAnnounce, &gotPeer.Left)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(testPeer, gotPeer) {
		t.Fatal(fixtureFailure("Existing peer incorrectly updated in the database", testPeer, gotPeer))
	}

	if gotStartTime != 1584996101 {
		t.Fatal(fixtureFailure("Start time incorrectly updated for existing peer", 1584996101, gotStartTime))
	}

	// Now test for new peer not in database
	testPeer = &cdb.Peer{
		UserID:       0,
		TorrentID:    2,
		Seeding:      true,
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
		Left:         65000,
	}

	db.RecordTransferHistory(testPeer, 0, 1000, 1, 0, 1, true)

	gotPeer = &cdb.Peer{
		UserID:    testPeer.UserID,
		TorrentID: testPeer.TorrentID,
	}

	for len(db.transferHistoryChannel) > 0 {
		time.Sleep(time.Second)
	}
	time.Sleep(200 * time.Millisecond)

	row = db.mainConn.sqlDb.QueryRow("SELECT seeding, starttime, last_announce, remaining "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", gotPeer.UserID, gotPeer.TorrentID)

	err = row.Scan(&gotPeer.Seeding, &gotPeer.StartTime, &gotPeer.LastAnnounce, &gotPeer.Left)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(testPeer, gotPeer) {
		t.Fatal(fixtureFailure("New peer incorrectly inserted in the database", testPeer, gotPeer))
	}
}

func TestRecordAndFlushTransferIP(t *testing.T) {
	prepareTestDatabase()

	testPeer := &cdb.Peer{
		UserID:       0,
		TorrentID:    0,
		ClientID:     1,
		Addr:         cdb.NewPeerAddressFromIPPort(net.IP{127, 0, 0, 1}, 63448),
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
	}

	var (
		initUpload    int64
		initDownload  int64
		upload        int64
		download      int64
		deltaDownload int64
		deltaUpload   int64
	)

	deltaDownload = 236
	deltaUpload = 3262

	row := db.mainConn.sqlDb.QueryRow("SELECT uploaded, downloaded "+
		"FROM transfer_ips WHERE uid = ? AND fid = ? AND ip = ? AND client_id = ?",
		testPeer.UserID, testPeer.TorrentID, testPeer.Addr.IPNumeric(), testPeer.ClientID)

	err := row.Scan(&initUpload, &initDownload)
	if err != nil {
		panic(err)
	}

	db.RecordTransferIP(testPeer, deltaUpload, deltaDownload)

	for len(db.transferIpsChannel) > 0 {
		time.Sleep(time.Second)
	}
	time.Sleep(200 * time.Millisecond)

	row = db.mainConn.sqlDb.QueryRow("SELECT uploaded, downloaded "+
		"FROM transfer_ips WHERE uid = ? AND fid = ? AND ip = ? AND client_id = ?",
		testPeer.UserID, testPeer.TorrentID, testPeer.Addr.IPNumeric(), testPeer.ClientID)

	err = row.Scan(&upload, &download)
	if err != nil {
		panic(err)
	}

	if initDownload+deltaDownload != download {
		t.Fatal(fixtureFailure(
			"Delta download incorrectly updated in the database",
			deltaDownload,
			download-initDownload,
		))
	}

	if initUpload+deltaUpload != upload {
		t.Fatal(fixtureFailure(
			"Delta upload incorrectly updated in the database",
			deltaUpload,
			upload-initUpload,
		))
	}

	// Check if existing peer being updated properly
	gotPeer := &cdb.Peer{
		UserID:    testPeer.UserID,
		TorrentID: testPeer.TorrentID,
		ClientID:  testPeer.ClientID,
		Addr:      cdb.NewPeerAddressFromIPPort(testPeer.Addr.IP(), 0),
		StartTime: testPeer.StartTime,
	}

	var gotStartTime int64

	row = db.mainConn.sqlDb.QueryRow("SELECT port, starttime, last_announce "+
		"FROM transfer_ips WHERE uid = ? AND fid = ? AND ip = ? AND client_id = ?",
		testPeer.UserID, testPeer.TorrentID, testPeer.Addr.IPNumeric(), testPeer.ClientID)

	var port uint16

	err = row.Scan(&port, &gotStartTime, &gotPeer.LastAnnounce)
	if err != nil {
		panic(err)
	}

	gotPeer.Addr = cdb.NewPeerAddressFromIPPort(gotPeer.Addr.IP(), port)

	if !reflect.DeepEqual(testPeer, gotPeer) {
		t.Fatal(fixtureFailure("Existing peer incorrectly updated in the database", testPeer, gotPeer))
	}

	if gotStartTime != 1584802402 {
		t.Fatal(fixtureFailure("Start time incorrectly updated for existing peer", 1584802402, gotStartTime))
	}

	// Now test for new peer not in database
	testPeer = &cdb.Peer{
		UserID:       1,
		TorrentID:    2,
		ClientID:     2,
		Addr:         cdb.NewPeerAddressFromIPPort(net.IP{127, 0, 0, 1}, 63448),
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
	}

	db.RecordTransferIP(testPeer, 0, 0)

	for len(db.transferIpsChannel) > 0 {
		time.Sleep(time.Second)
	}
	time.Sleep(200 * time.Millisecond)

	gotPeer = &cdb.Peer{
		UserID:    testPeer.UserID,
		TorrentID: testPeer.TorrentID,
		ClientID:  testPeer.ClientID,
		Addr:      cdb.NewPeerAddressFromIPPort(testPeer.Addr.IP(), 0),
	}

	row = db.mainConn.sqlDb.QueryRow("SELECT port, starttime, last_announce "+
		"FROM transfer_ips WHERE uid = ? AND fid = ? AND ip = ? AND client_id = ?",
		testPeer.UserID, testPeer.TorrentID, testPeer.Addr.IPNumeric(), testPeer.ClientID)

	err = row.Scan(&port, &gotPeer.StartTime, &gotPeer.LastAnnounce)
	if err != nil {
		panic(err)
	}

	gotPeer.Addr = cdb.NewPeerAddressFromIPPort(gotPeer.Addr.IP(), port)

	if !reflect.DeepEqual(testPeer, gotPeer) {
		t.Fatal(fixtureFailure("New peer is incorrectly inserted in the database", testPeer, gotPeer))
	}
}

func TestRecordAndFlushSnatch(t *testing.T) {
	prepareTestDatabase()

	testPeer := &cdb.Peer{
		UserID:    1,
		TorrentID: 1,
	}

	var (
		snatchTime int64
		currTime   int64
	)

	currTime = time.Now().Unix()

	db.RecordSnatch(testPeer, currTime)

	for len(db.snatchChannel) > 0 {
		time.Sleep(time.Second)
	}
	time.Sleep(200 * time.Millisecond)

	row := db.mainConn.sqlDb.QueryRow("SELECT snatched_time "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", testPeer.UserID, testPeer.TorrentID)

	err := row.Scan(&snatchTime)
	if err != nil {
		panic(err)
	}

	if snatchTime != currTime {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Snatches incorrectly updated in the database for torrentId %v", 1),
			currTime,
			snatchTime))
	}
}

func TestRecordAndFlushTorrents(t *testing.T) {
	prepareTestDatabase()

	h := cdb.TorrentHash{114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56}
	torrent := db.Torrents[h]
	torrent.LastAction = time.Now().Unix()
	torrent.Seeders[cdb.NewPeerKey(1, cdb.PeerIDFromRawString("test_peer_id_num_one"))] = &cdb.Peer{
		UserID:       1,
		TorrentID:    torrent.ID,
		ClientID:     1,
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
	}
	torrent.Leechers[cdb.NewPeerKey(3, cdb.PeerIDFromRawString("test_peer_id_num_two"))] = &cdb.Peer{
		UserID:       3,
		TorrentID:    torrent.ID,
		ClientID:     2,
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
	}

	db.RecordTorrent(torrent, 5)

	for len(db.torrentChannel) > 0 {
		time.Sleep(time.Second)
	}
	time.Sleep(200 * time.Millisecond)

	var (
		snatched    uint16
		lastAction  int64
		numLeechers int
		numSeeders  int
	)

	row := db.mainConn.sqlDb.QueryRow("SELECT Snatched, last_action, Seeders, Leechers "+
		"FROM torrents WHERE ID = ?", torrent.ID)
	err := row.Scan(&snatched, &lastAction, &numSeeders, &numLeechers)

	if err != nil {
		panic(err)
	}

	if torrent.Snatched+5 != snatched {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Snatches incorrectly updated in the database for torrent %x", h),
			torrent.Snatched+5,
			snatched,
		))
	}

	if torrent.LastAction != lastAction {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Last incorrectly updated in the database for torrent %x", h),
			torrent.LastAction,
			lastAction,
		))
	}

	if len(torrent.Seeders) != numSeeders {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Seeders incorrectly updated in the database for torrent %x", h),
			len(torrent.Seeders),
			numSeeders,
		))
	}

	if len(torrent.Leechers) != numLeechers {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Leechers incorrectly updated in the database for torrent %x", h),
			len(torrent.Leechers),
			numLeechers,
		))
	}
}

func TestTerminate(_ *testing.T) {
	prepareTestDatabase()

	db.Terminate()

	db.Init() // Restart for other tests
}

func prepareTestDatabase() {
	if err := fixtures.Load(); err != nil {
		panic(err)
	}
}

func fixtureFailure(msg string, expected interface{}, got interface{}) string {
	return fmt.Sprintf("%s\nExpected: %+v\nGot: %+v", msg, expected, got)
}

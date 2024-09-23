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
	"math"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	cdb "chihaya/database/types"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/google/go-cmp/cmp"
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
		testfixtures.Database(db.conn),
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

	dbUsers := make(map[string]*cdb.User)
	db.Users.Store(&dbUsers)

	testUser1 := &cdb.User{}
	testUser1.ID.Store(1)
	testUser1.DownMultiplier.Store(math.Float64bits(1))
	testUser1.UpMultiplier.Store(math.Float64bits(1))
	testUser1.DisableDownload.Store(false)
	testUser1.TrackerHide.Store(false)

	testUser2 := &cdb.User{}
	testUser2.ID.Store(2)
	testUser2.DownMultiplier.Store(math.Float64bits(2))
	testUser2.UpMultiplier.Store(math.Float64bits(0.5))
	testUser2.DisableDownload.Store(true)
	testUser2.TrackerHide.Store(true)

	users := map[string]*cdb.User{
		"mUztWMpBYNCqzmge6vGeEUGSrctJbgpQ": testUser1,
		"tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw": testUser2,
	}

	// Test with fresh data
	db.loadUsers()

	dbUsers = *db.Users.Load()

	if len(dbUsers) != len(users) {
		t.Fatal(fixtureFailure("Did not load all users as expected from fixture file", len(users), len(dbUsers)))
	}

	for passkey, user := range users {
		if !reflect.DeepEqual(user, dbUsers[passkey]) {
			t.Fatal(fixtureFailure(
				fmt.Sprintf("Did not load user (%s) as expected from fixture file", passkey),
				user,
				dbUsers[passkey]))
		}
	}

	// Now test load on top of existing data
	oldUsers := dbUsers

	db.loadUsers()

	dbUsers = *db.Users.Load()

	if !reflect.DeepEqual(oldUsers, dbUsers) {
		t.Fatal(fixtureFailure("Did not reload users as expected from fixture file", oldUsers, dbUsers))
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

	dbTorrents := make(map[cdb.TorrentHash]*cdb.Torrent)
	db.Torrents.Store(&dbTorrents)

	t1 := &cdb.Torrent{
		Seeders:  map[cdb.PeerKey]*cdb.Peer{},
		Leechers: map[cdb.PeerKey]*cdb.Peer{},
	}
	t1.ID.Store(1)
	t1.Status.Store(1)
	t1.Snatched.Store(2)
	t1.DownMultiplier.Store(math.Float64bits(1))
	t1.UpMultiplier.Store(math.Float64bits(1))
	t1.Group.GroupID.Store(1)
	t1.Group.TorrentType.Store(cdb.MustTorrentTypeFromString("anime"))

	t2 := &cdb.Torrent{
		Seeders:  map[cdb.PeerKey]*cdb.Peer{},
		Leechers: map[cdb.PeerKey]*cdb.Peer{},
	}
	t2.ID.Store(2)
	t2.Status.Store(0)
	t2.Snatched.Store(0)
	t2.DownMultiplier.Store(math.Float64bits(2))
	t2.UpMultiplier.Store(math.Float64bits(0.5))
	t2.Group.GroupID.Store(1)
	t2.Group.TorrentType.Store(cdb.MustTorrentTypeFromString("music"))

	t3 := &cdb.Torrent{
		Seeders:  map[cdb.PeerKey]*cdb.Peer{},
		Leechers: map[cdb.PeerKey]*cdb.Peer{},
	}
	t3.ID.Store(3)
	t3.Status.Store(0)
	t3.Snatched.Store(0)
	t3.DownMultiplier.Store(math.Float64bits(1))
	t3.UpMultiplier.Store(math.Float64bits(1))
	t3.Group.GroupID.Store(2)
	t3.Group.TorrentType.Store(cdb.MustTorrentTypeFromString("anime"))

	torrents := map[cdb.TorrentHash]*cdb.Torrent{
		{114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56}: t1,
		{22, 168, 45, 221, 87, 225, 140, 177, 94, 34, 242, 225, 196, 234, 222, 46, 187, 131, 177, 155}:    t2,
		{89, 252, 84, 49, 177, 28, 118, 28, 148, 205, 62, 185, 8, 37, 234, 110, 109, 200, 165, 241}:       t3,
	}

	// Test with fresh data
	db.loadTorrents()

	dbTorrents = *db.Torrents.Load()

	if len(dbTorrents) != len(torrents) {
		t.Fatal(fixtureFailure("Did not load all torrents as expected from fixture file",
			len(torrents),
			len(dbTorrents)))
	}

	for hash, torrent := range torrents {
		if !cmp.Equal(torrent, dbTorrents[hash], cdb.TorrentTestCompareOptions...) {
			hashHex, _ := hash.MarshalText()
			t.Fatal(fixtureFailure(
				fmt.Sprintf("Did not load torrent (%s) as expected from fixture file", string(hashHex)),
				torrent,
				dbTorrents[hash]))
		}
	}

	// Now test load on top of existing data
	oldTorrents := dbTorrents

	db.loadTorrents()

	dbTorrents = *db.Torrents.Load()

	if !cmp.Equal(oldTorrents, dbTorrents, cdb.TorrentTestCompareOptions...) {
		t.Fatal(fixtureFailure("Did not reload torrents as expected from fixture file", oldTorrents, dbTorrents))
	}
}

func TestLoadGroupsFreeleech(t *testing.T) {
	prepareTestDatabase()

	dbMap := make(map[cdb.TorrentGroupKey]*cdb.TorrentGroupFreeleech)
	db.TorrentGroupFreeleech.Store(&dbMap)

	torrentGroupFreeleech := map[cdb.TorrentGroupKey]*cdb.TorrentGroupFreeleech{
		cdb.MustTorrentGroupKeyFromString("anime", 2): {
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

	dbTorrents := *db.Torrents.Load()

	h := cdb.TorrentHash{114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56}
	dbTorrent := dbTorrents[h]

	torrent := cdb.Torrent{
		Seeders:  dbTorrent.Seeders,
		Leechers: dbTorrent.Leechers,
	}
	torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
	torrent.LeechersLength.Store(uint32(len(torrent.Leechers)))
	torrent.ID.Store(dbTorrent.ID.Load())
	torrent.Status.Store(dbTorrent.Status.Load())
	torrent.Snatched.Store(dbTorrent.Snatched.Load())
	torrent.DownMultiplier.Store(dbTorrent.DownMultiplier.Load())
	torrent.UpMultiplier.Store(dbTorrent.UpMultiplier.Load())
	torrent.Group.GroupID.Store(dbTorrent.Group.GroupID.Load())
	torrent.Group.TorrentType.Store(dbTorrent.Group.TorrentType.Load())

	torrent.Status.Store(0)

	db.UnPrune(dbTorrents[h])

	db.loadTorrents()

	dbTorrents = *db.Torrents.Load()

	if !cmp.Equal(&torrent, dbTorrents[h], cdb.TorrentTestCompareOptions...) {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Torrent (%x) was not unpruned properly", h),
			&torrent,
			dbTorrents[h]))
	}
}

func TestRecordAndFlushUsers(t *testing.T) {
	prepareTestDatabase()

	dbUsers := *db.Users.Load()

	testUser := dbUsers["tbHfQDQ9xDaQdsNv5CZBtHPfk7KGzaCw"]

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
	deltaDownload = int64(float64(deltaRawDownload) * math.Float64frombits(testUser.DownMultiplier.Load()))
	deltaUpload = int64(float64(deltaRawUpload) * math.Float64frombits(testUser.UpMultiplier.Load()))

	row := db.conn.QueryRow("SELECT Uploaded, Downloaded, rawup, rawdl "+
		"FROM users_main WHERE ID = ?", testUser.ID.Load())

	err := row.Scan(&initUpload, &initDownload, &initRawUpload, &initRawDownload)
	if err != nil {
		panic(err)
	}

	db.QueueUser(testUser, deltaRawUpload, deltaRawDownload, deltaUpload, deltaDownload)

	for len(db.userChannel) > 0 {
		time.Sleep(time.Second)
	}

	time.Sleep(200 * time.Millisecond)

	row = db.conn.QueryRow("SELECT Uploaded, Downloaded, rawup, rawdl "+
		"FROM users_main WHERE ID = ?", testUser.ID.Load())

	err = row.Scan(&upload, &download, &rawUpload, &rawDownload)
	if err != nil {
		panic(err)
	}

	if initDownload+deltaDownload != download {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Delta download incorrectly updated in the database for user %v", testUser.ID.Load()),
			deltaDownload,
			download-initDownload,
		))
	}

	if initUpload+deltaUpload != upload {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Delta upload incorrectly updated in the database for user %v", testUser.ID.Load()),
			deltaUpload,
			upload-initUpload,
		))
	}

	if initRawDownload+deltaRawDownload != rawDownload {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Delta raw download incorrectly updated in the database for user %v", testUser.ID.Load()),
			deltaRawDownload,
			rawDownload-initRawDownload,
		))
	}

	if initRawUpload+deltaRawUpload != rawUpload {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Delta raw upload incorrectly updated in the database for user %v", testUser.ID.Load()),
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

	row := db.conn.QueryRow("SELECT uploaded, downloaded, activetime, seedtime, active, snatched "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", testPeer.UserID, testPeer.TorrentID)

	err := row.Scan(&initRawUpload, &initRawDownload, &initActiveTime, &initSeedTime, &initActive, &initSnatch)
	if err != nil {
		panic(err)
	}

	db.QueueTransferHistory(testPeer,
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

	row = db.conn.QueryRow("SELECT uploaded, downloaded, activetime, seedtime, active, snatched "+
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

	row = db.conn.QueryRow("SELECT seeding, starttime, last_announce, remaining "+
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

	db.QueueTransferHistory(testPeer, 0, 1000, 1, 0, 1, true)

	gotPeer = &cdb.Peer{
		UserID:    testPeer.UserID,
		TorrentID: testPeer.TorrentID,
	}

	for len(db.transferHistoryChannel) > 0 {
		time.Sleep(time.Second)
	}

	time.Sleep(200 * time.Millisecond)

	row = db.conn.QueryRow("SELECT seeding, starttime, last_announce, remaining "+
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

	row := db.conn.QueryRow("SELECT uploaded, downloaded "+
		"FROM transfer_ips WHERE uid = ? AND fid = ? AND ip = ? AND client_id = ?",
		testPeer.UserID, testPeer.TorrentID, testPeer.Addr.IPNumeric(), testPeer.ClientID)

	err := row.Scan(&initUpload, &initDownload)
	if err != nil {
		panic(err)
	}

	db.QueueTransferIP(testPeer, testPeer.Addr, deltaUpload, deltaDownload)

	for len(db.transferIpsChannel) > 0 {
		time.Sleep(time.Second)
	}

	time.Sleep(200 * time.Millisecond)

	row = db.conn.QueryRow("SELECT uploaded, downloaded "+
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

	row = db.conn.QueryRow("SELECT port, starttime, last_announce "+
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

	db.QueueTransferIP(testPeer, testPeer.Addr, 0, 0)

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

	row = db.conn.QueryRow("SELECT port, starttime, last_announce "+
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
		snatchTime = time.Now()
		recordTime int64
	)

	db.QueueSnatch(testPeer, snatchTime.Unix())

	for len(db.snatchChannel) > 0 {
		time.Sleep(time.Second)
	}

	time.Sleep(200 * time.Millisecond)

	row := db.conn.QueryRow("SELECT snatched_time "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", testPeer.UserID, testPeer.TorrentID)

	err := row.Scan(&recordTime)
	if err != nil {
		panic(err)
	}

	if recordTime != snatchTime.Unix() {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Snatch time incorrectly updated in the database for torrent %v", 1),
			recordTime,
			snatchTime))
	}

	db.QueueSnatch(testPeer, snatchTime.Add(time.Second*1337).Unix())

	for len(db.snatchChannel) > 0 {
		time.Sleep(time.Second)
	}

	row = db.conn.QueryRow("SELECT snatched_time "+
		"FROM transfer_history WHERE uid = ? AND fid = ?", testPeer.UserID, testPeer.TorrentID)

	err = row.Scan(&recordTime)
	if err != nil {
		panic(err)
	}

	if recordTime != snatchTime.Unix() {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Snatch time was incorrectly updated in the database for torrent %v", 1),
			recordTime,
			snatchTime))
	}
}

func TestRecordAndFlushTorrents(t *testing.T) {
	prepareTestDatabase()

	h := cdb.TorrentHash{114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56}
	torrent := (*db.Torrents.Load())[h]
	torrent.LastAction.Store(time.Now().Unix())
	torrent.Seeders[cdb.NewPeerKey(1, cdb.PeerIDFromRawString("test_peer_id_num_one"))] = &cdb.Peer{
		UserID:       1,
		TorrentID:    torrent.ID.Load(),
		ClientID:     1,
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
	}
	torrent.Leechers[cdb.NewPeerKey(3, cdb.PeerIDFromRawString("test_peer_id_num_two"))] = &cdb.Peer{
		UserID:       3,
		TorrentID:    torrent.ID.Load(),
		ClientID:     2,
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
	}
	torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))
	torrent.LeechersLength.Store(uint32(len(torrent.Leechers)))

	db.QueueTorrent(torrent, 5)

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

	row := db.conn.QueryRow("SELECT Snatched, last_action, Seeders, Leechers "+
		"FROM torrents WHERE ID = ?", torrent.ID.Load())

	err := row.Scan(&snatched, &lastAction, &numSeeders, &numLeechers)
	if err != nil {
		panic(err)
	}

	if uint16(torrent.Snatched.Load())+5 != snatched {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Snatches incorrectly updated in the database for torrent %x", h),
			torrent.Snatched.Load()+5,
			snatched,
		))
	}

	if torrent.LastAction.Load() != lastAction {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("Last incorrectly updated in the database for torrent %x", h),
			torrent.LastAction.Load(),
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

	if int(torrent.SeedersLength.Load()) != numSeeders {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("SeedersLength incorrectly updated in the database for torrent %x", h),
			len(torrent.Seeders),
			numSeeders,
		))
	}

	if int(torrent.LeechersLength.Load()) != numLeechers {
		t.Fatal(fixtureFailure(
			fmt.Sprintf("LeechersLength incorrectly updated in the database for torrent %x", h),
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

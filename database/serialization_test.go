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
	"math"
	"net/netip"
	"reflect"
	"testing"
	"time"

	cdb "chihaya/database/types"

	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
)

func TestSerializer(t *testing.T) {
	testTorrents := make(map[cdb.TorrentHash]*cdb.Torrent)
	testUsers := make(map[string]*cdb.User)

	testUser := &cdb.User{}
	testUser.ID.Store(12)
	testUser.DownMultiplier.Store(math.Float64bits(1))
	testUser.UpMultiplier.Store(math.Float64bits(1))
	testUser.DisableDownload.Store(false)
	testUser.TrackerHide.Store(false)

	testUsers["mUztWMpBYNCqzmge6vGeEUGSrctJbgpQ"] = testUser

	testPeer := &cdb.Peer{
		UserID:       12,
		TorrentID:    10,
		ClientID:     4,
		Addr:         cdb.NewPeerAddressFromAddrPort(netip.AddrFrom4([4]byte{127, 0, 0, 1}), 63448),
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
		Seeding:      true,
		Left:         0,
		Uploaded:     100,
		Downloaded:   1000,
		ID:           cdb.PeerIDFromRawString("12-10-2130706433-4"),
	}

	testTorrentHash := cdb.TorrentHash{
		114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56,
	}

	torrent := &cdb.Torrent{
		Seeders: map[cdb.PeerKey]*cdb.Peer{
			cdb.NewPeerKey(12, cdb.PeerIDFromRawString("peer_is_twenty_chars")): testPeer,
		},
		Leechers: map[cdb.PeerKey]*cdb.Peer{},
	}
	torrent.ID.Store(10)
	torrent.Status.Store(1)
	torrent.Snatched.Store(100)
	torrent.LastAction.Store(time.Now().Unix())
	torrent.DownMultiplier.Store(math.Float64bits(1))
	torrent.UpMultiplier.Store(math.Float64bits(1))
	torrent.SeedersLength.Store(uint32(len(torrent.Seeders)))

	torrent.Group.GroupID.Store(1)
	torrent.Group.TorrentType.Store(cdb.MustTorrentTypeFromString("anime"))

	testTorrents[testTorrentHash] = torrent

	// Prepare empty map to populate with test data
	dbTorrents := make(map[cdb.TorrentHash]*cdb.Torrent)
	db.Torrents.Store(&dbTorrents)

	dbUsers := make(map[string]*cdb.User)
	db.Users.Store(&dbUsers)

	if err := copier.Copy(&dbTorrents, testTorrents); err != nil {
		panic(err)
	}

	if err := copier.Copy(&dbUsers, testUsers); err != nil {
		panic(err)
	}

	db.serialize()

	// Reset map to fully test deserialization
	dbTorrents = make(map[cdb.TorrentHash]*cdb.Torrent)
	db.Torrents.Store(&dbTorrents)

	dbUsers = make(map[string]*cdb.User)
	db.Users.Store(&dbUsers)

	db.deserialize()

	dbTorrents = *db.Torrents.Load()
	dbUsers = *db.Users.Load()

	if !cmp.Equal(dbTorrents, testTorrents, cdb.TorrentTestCompareOptions...) {
		t.Fatalf("Torrents (%v) after serialization and deserialization do not match original torrents (%v)!",
			dbTorrents, testTorrents)
	}

	if !reflect.DeepEqual(dbUsers, testUsers) {
		t.Fatalf("Users (%v) after serialization and deserialization do not match original users (%v)!",
			dbUsers, testUsers)
	}
}

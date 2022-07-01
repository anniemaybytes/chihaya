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
	"chihaya/database/types"
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/copier"
)

func TestSerializer(t *testing.T) {
	testTorrents := make(map[string]*types.Torrent)
	testUsers := make(map[string]*types.User)

	testUsers["mUztWMpBYNCqzmge6vGeEUGSrctJbgpQ"] = &types.User{
		DisableDownload: false,
		TrackerHide:     false,
		ID:              12,
		UpMultiplier:    1,
		DownMultiplier:  1,
	}

	testPeer := &types.Peer{
		UserID:       12,
		TorrentID:    10,
		ClientID:     4,
		IP:           2130706433,
		Port:         63448,
		StartTime:    time.Now().Unix(),
		LastAnnounce: time.Now().Unix(),
		Seeding:      true,
		Left:         0,
		Uploaded:     100,
		Downloaded:   1000,
		ID:           "12-10-2130706433-4",
	}

	testTorrentHash := []byte{
		114, 239, 32, 237, 220, 181, 67, 143, 115, 182, 216, 141, 120, 196, 223, 193, 102, 123, 137, 56,
	}
	testTorrents[string(testTorrentHash)] = &types.Torrent{
		Status:         1,
		Snatched:       100,
		ID:             10,
		LastAction:     time.Now().Unix(),
		UpMultiplier:   1,
		DownMultiplier: 1,
		Seeders:        map[string]*types.Peer{"12-peer_is_twenty_chars": testPeer},
	}

	db.Torrents = make(map[string]*types.Torrent)
	db.Users = make(map[string]*types.User)

	if err := copier.Copy(&db.Torrents, testTorrents); err != nil {
		panic(err)
	}

	if err := copier.Copy(&db.Users, testUsers); err != nil {
		panic(err)
	}

	db.serialize()
	db.deserialize()

	if !reflect.DeepEqual(db.Torrents, testTorrents) {
		t.Fatalf("Torrents after serialization and deserialization do not match original torrents!")
	}

	if !reflect.DeepEqual(db.Users, testUsers) {
		t.Fatalf("Users after serialization and deserialization do not match original users!")
	}
}

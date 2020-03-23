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

package types

type Peer struct {
	Seeding      bool
	ClientID     uint16
	Port         uint16
	UserID       uint32
	IP           uint32
	TorrentID    uint32
	Uploaded     uint64
	Downloaded   uint64
	Left         uint64
	StartTime    int64 // unix time
	LastAnnounce int64
	ID           string
	IPAddr       string
	Addr         []byte
}

type Torrent struct {
	Status         uint8
	Snatched       uint16
	ID             uint32
	LastAction     int64 // unix time
	UpMultiplier   float64
	DownMultiplier float64

	Seeders  map[string]*Peer
	Leechers map[string]*Peer
}

type User struct {
	DisableDownload bool
	TrackerHide     bool
	ID              uint32
	UpMultiplier    float64
	DownMultiplier  float64
}

type UserTorrentPair struct {
	UserID    uint32
	TorrentID uint32
}

var (
	TorrentCacheFile = "torrent-cache"
	UserCacheFile    = "user-cache"
)

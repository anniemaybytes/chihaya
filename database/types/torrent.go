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

type Torrent struct {
	Seeders  map[string]*Peer
	Leechers map[string]*Peer

	Group TorrentGroup
	ID    uint32

	Snatched uint16

	Status     uint8
	LastAction int64 // unix time

	UpMultiplier   float64
	DownMultiplier float64
}

type TorrentGroupFreeleech struct {
	UpMultiplier   float64
	DownMultiplier float64
}

type TorrentGroup struct {
	TorrentType string
	GroupID     uint32
}

// TorrentCacheFile holds filename used by serializer for this type
var TorrentCacheFile = "torrent-cache"

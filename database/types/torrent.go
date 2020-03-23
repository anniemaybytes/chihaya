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
	Status         uint8
	Snatched       uint16
	ID             uint32
	LastAction     int64 // unix time
	UpMultiplier   float64
	DownMultiplier float64

	Seeders  map[string]*Peer
	Leechers map[string]*Peer
}

var TorrentCacheFile = "torrent-cache"

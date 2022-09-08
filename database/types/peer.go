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
	ID string

	IPAddr string
	Addr   []byte
	IP     uint32
	Port   uint16

	Uploaded   uint64
	Downloaded uint64
	Left       uint64

	StartTime    int64 // unix time
	LastAnnounce int64

	TorrentID uint32
	UserID    uint32
	ClientID  uint16

	Seeding bool
}

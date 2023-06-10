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

import (
	"encoding/binary"
	"math"
)

type User struct {
	ID uint32

	DisableDownload bool

	TrackerHide bool

	UpMultiplier   float64
	DownMultiplier float64
}

func (u *User) Load(reader readerAndByteReader) (err error) {
	if err = binary.Read(reader, binary.LittleEndian, &u.ID); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &u.DisableDownload); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &u.TrackerHide); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &u.UpMultiplier); err != nil {
		return err
	}

	return binary.Read(reader, binary.LittleEndian, &u.DownMultiplier)
}

func (u *User) Append(preAllocatedBuffer []byte) (buf []byte) {
	buf = preAllocatedBuffer
	buf = binary.LittleEndian.AppendUint32(buf, u.ID)

	if u.DisableDownload {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	if u.TrackerHide {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	buf = binary.LittleEndian.AppendUint64(buf, math.Float64bits(u.UpMultiplier))
	buf = binary.LittleEndian.AppendUint64(buf, math.Float64bits(u.DownMultiplier))

	return buf
}

type UserTorrentPair struct {
	UserID    uint32
	TorrentID uint32
}

// UserCacheFile holds filename used by serializer for this type
var UserCacheFile = "user-cache"

// UserCacheVersion Used to distinguish old versions on the on-disk cache.
// Bump when fields are altered on User struct
const UserCacheVersion = 1

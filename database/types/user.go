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
	"encoding/json"
	"math"
	"sync/atomic"
)

type User struct {
	ID atomic.Uint32

	DisableDownload atomic.Bool

	TrackerHide atomic.Bool

	// UpMultiplier A float64 under the covers
	UpMultiplier atomic.Uint64
	// DownMultiplier A float64 under the covers
	DownMultiplier atomic.Uint64
}

func (u *User) Load(_ uint64, reader readerAndByteReader) (err error) {
	var (
		id                           uint32
		disableDownload, trackerHide bool
		upMultiplier, downMultiplier float64
	)

	if err = binary.Read(reader, binary.LittleEndian, &id); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &disableDownload); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &trackerHide); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &upMultiplier); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &downMultiplier); err != nil {
		return err
	}

	u.ID.Store(id)
	u.DisableDownload.Store(disableDownload)
	u.TrackerHide.Store(trackerHide)
	u.UpMultiplier.Store(math.Float64bits(upMultiplier))
	u.DownMultiplier.Store(math.Float64bits(downMultiplier))

	return nil
}

func (u *User) Append(preAllocatedBuffer []byte) (buf []byte) {
	buf = preAllocatedBuffer
	buf = binary.LittleEndian.AppendUint32(buf, u.ID.Load())

	if u.DisableDownload.Load() {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	if u.TrackerHide.Load() {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	buf = binary.LittleEndian.AppendUint64(buf, u.UpMultiplier.Load())
	buf = binary.LittleEndian.AppendUint64(buf, u.DownMultiplier.Load())

	return buf
}

var encodeJSONUserMap = make(map[string]any)

// MarshalJSON Due to using atomics, JSON will not marshal values within them.
// This is only safe to call from a single thread at once
func (u *User) MarshalJSON() (buf []byte, err error) {
	encodeJSONUserMap["ID"] = u.ID.Load()
	encodeJSONUserMap["DisableDownload"] = u.DisableDownload.Load()
	encodeJSONUserMap["TrackerHide"] = u.TrackerHide.Load()
	encodeJSONUserMap["UpMultiplier"] = math.Float64frombits(u.UpMultiplier.Load())
	encodeJSONUserMap["DownMultiplier"] = math.Float64frombits(u.UpMultiplier.Load())

	return json.Marshal(encodeJSONUserMap)
}

type decodeJSONUser struct {
	ID              uint32
	DisableDownload bool
	TrackerHide     bool
	UpMultiplier    float64
	DownMultiplier  float64
}

// UnmarshalJSON Due to using atomics, JSON will not marshal values within them.
// This is only safe to call from a single thread at once
func (u *User) UnmarshalJSON(buf []byte) (err error) {
	var userJSON decodeJSONUser
	if err = json.Unmarshal(buf, &userJSON); err != nil {
		return err
	}

	u.ID.Store(userJSON.ID)
	u.DisableDownload.Store(userJSON.DisableDownload)
	u.TrackerHide.Store(userJSON.TrackerHide)
	u.UpMultiplier.Store(math.Float64bits(userJSON.UpMultiplier))
	u.DownMultiplier.Store(math.Float64bits(userJSON.DownMultiplier))

	return nil
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

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
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"sync"
)

const TorrentHashSize = 20

// TorrentHash SHA-1 hash (20 bytes)
type TorrentHash [TorrentHashSize]byte

var errWrongHashSize = errors.New("wrong hash size")
var errNilHash = errors.New("nil hash")
var errInvalidType = errors.New("invalid type")

func TorrentHashFromBytes(buf []byte) (h TorrentHash) {
	if len(buf) != TorrentHashSize {
		return
	}

	copy(h[:], buf)

	return h
}

//goland:noinspection GoMixedReceiverTypes
func (h *TorrentHash) Scan(src any) error {
	if src == nil {
		return nil
	} else if buf, ok := src.([]byte); ok {
		if len(buf) == 0 {
			return errNilHash
		}
		if len(buf) != TorrentHashSize {
			return errWrongHashSize
		}
		copy((*h)[:], buf)

		return nil
	}

	return errInvalidType
}

//goland:noinspection GoMixedReceiverTypes
func (h *TorrentHash) Value() (driver.Value, error) {
	return (*h)[:], nil
}

//goland:noinspection GoMixedReceiverTypes
func (h TorrentHash) MarshalJSON() ([]byte, error) {
	var buf [TorrentHashSize*2 + 2]byte
	buf[0] = '"'
	buf[TorrentHashSize*2+1] = '"'
	hex.Encode(buf[1:], h[:])

	return buf[:], nil
}

//goland:noinspection GoMixedReceiverTypes
func (h *TorrentHash) UnmarshalJSON(b []byte) error {
	if len(b) != TorrentHashSize*2+2 {
		return errWrongHashSize
	}

	if _, err := hex.Decode(h[:], b[1:len(b)-1]); err != nil {
		return err
	}

	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (h TorrentHash) MarshalText() ([]byte, error) {
	var buf [TorrentHashSize * 2]byte

	hex.Encode(buf[:], h[:])

	return buf[:], nil
}

//goland:noinspection GoMixedReceiverTypes
func (h *TorrentHash) UnmarshalText(b []byte) error {
	if len(b) != TorrentHashSize*2 {
		return errWrongHashSize
	}

	if _, err := hex.Decode(h[:], b[:]); err != nil {
		return err
	}

	return nil
}

type Torrent struct {
	Seeders  map[PeerKey]*Peer
	Leechers map[PeerKey]*Peer

	Group TorrentGroup
	ID    uint32

	Snatched uint16

	Status     uint8
	LastAction int64 // unix time

	UpMultiplier   float64
	DownMultiplier float64

	// lock This must be taken whenever read or write is made to fields on this torrent.
	// Maybe single sync.Mutex is easier to handle, but prevent concurrent access.
	lock sync.RWMutex
}

func (t *Torrent) Lock() {
	t.lock.Lock()
}

func (t *Torrent) Unlock() {
	t.lock.Unlock()
}

func (t *Torrent) RLock() {
	t.lock.RLock()
}

func (t *Torrent) RUnlock() {
	t.lock.RUnlock()
}

func (t *Torrent) Load(version uint64, reader readerAndByteReader) (err error) {
	var varIntLen uint64

	if varIntLen, err = binary.ReadUvarint(reader); err != nil {
		return err
	}

	t.Seeders = make(map[PeerKey]*Peer, varIntLen)

	var k PeerKey
	for i := uint64(0); i < varIntLen; i++ {
		if _, err = io.ReadFull(reader, k[:]); err != nil {
			return err
		}

		s := &Peer{}

		if err = s.Load(version, reader); err != nil {
			return err
		}

		t.Seeders[k] = s
	}

	if varIntLen, err = binary.ReadUvarint(reader); err != nil {
		return err
	}

	t.Leechers = make(map[PeerKey]*Peer, varIntLen)

	for i := uint64(0); i < varIntLen; i++ {
		if _, err = io.ReadFull(reader, k[:]); err != nil {
			return err
		}

		l := &Peer{}

		if err = l.Load(version, reader); err != nil {
			return err
		}

		t.Leechers[k] = l
	}

	if err = t.Group.Load(version, reader); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &t.ID); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &t.Snatched); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &t.Status); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &t.LastAction); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &t.UpMultiplier); err != nil {
		return err
	}

	return binary.Read(reader, binary.LittleEndian, &t.DownMultiplier)
}

func (t *Torrent) Append(preAllocatedBuffer []byte) (buf []byte) {
	t.RLock()
	defer t.RUnlock()

	buf = preAllocatedBuffer
	buf = binary.AppendUvarint(buf, uint64(len(t.Seeders)))

	for k, s := range t.Seeders {
		buf = append(buf, k[:]...)

		buf = s.Append(buf)
	}

	buf = binary.AppendUvarint(buf, uint64(len(t.Leechers)))

	for k, l := range t.Leechers {
		buf = append(buf, k[:]...)

		buf = l.Append(buf)
	}

	buf = t.Group.Append(buf)

	buf = binary.LittleEndian.AppendUint32(buf, t.ID)
	buf = binary.LittleEndian.AppendUint16(buf, t.Snatched)
	buf = append(buf, t.Status)
	buf = binary.LittleEndian.AppendUint64(buf, uint64(t.LastAction))
	buf = binary.LittleEndian.AppendUint64(buf, math.Float64bits(t.UpMultiplier))
	buf = binary.LittleEndian.AppendUint64(buf, math.Float64bits(t.DownMultiplier))

	return buf
}

type TorrentGroupFreeleech struct {
	UpMultiplier   float64
	DownMultiplier float64
}

type TorrentGroup struct {
	TorrentType string
	GroupID     uint32
}

func (g *TorrentGroup) Load(_ uint64, reader readerAndByteReader) (err error) {
	var varIntLen uint64

	if varIntLen, err = binary.ReadUvarint(reader); err != nil {
		return err
	}

	buf := make([]byte, varIntLen)

	if _, err = io.ReadFull(reader, buf); err != nil {
		return err
	}

	g.TorrentType = string(buf)

	return binary.Read(reader, binary.LittleEndian, &g.GroupID)
}

func (g *TorrentGroup) Append(preAllocatedBuffer []byte) (buf []byte) {
	buf = preAllocatedBuffer
	buf = binary.AppendUvarint(buf, uint64(len(g.TorrentType)))
	buf = append(buf, []byte(g.TorrentType)...)

	return binary.LittleEndian.AppendUint32(buf, g.GroupID)
}

// TorrentCacheFile holds filename used by serializer for this type
var TorrentCacheFile = "torrent-cache"

// TorrentCacheVersion Used to distinguish old versions on the on-disk cache.
// Bump when fields are altered on Torrent, Peer or TorrentGroup structs
const TorrentCacheVersion = 2

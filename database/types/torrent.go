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
	"encoding/json"
	"errors"
	"io"
	"math"
	"sync"
	"sync/atomic"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

	// SeedersLength Contains the length of Seeders. When Seeders is modified this field must be updated
	SeedersLength atomic.Uint32
	// LeechersLength Contains the length of Leechers. When LeechersLength is modified this field must be updated
	LeechersLength atomic.Uint32

	Group TorrentGroup
	ID    atomic.Uint32

	// Snatched 16 bits
	Snatched atomic.Uint32

	// Snatched 8 bits
	Status atomic.Uint32
	// LastAction UNIX time
	LastAction atomic.Int64

	// peerLock This must be taken whenever read or write is made to Seeders or Leechers fields on this torrent
	peerLock sync.Mutex

	// UpMultiplier float64
	UpMultiplier atomic.Uint64
	// DownMultiplier float64
	DownMultiplier atomic.Uint64
}

func (t *Torrent) PeerLock() {
	t.peerLock.Lock()
}

func (t *Torrent) PeerUnlock() {
	t.peerLock.Unlock()
}

func (t *Torrent) Load(version uint64, reader readerAndByteReader) (err error) {
	var (
		id                           uint32
		snatched                     uint16
		status                       uint8
		lastAction                   int64
		upMultiplier, downMultiplier float64
	)

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

	t.SeedersLength.Store(uint32(len(t.Seeders)))

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

	t.LeechersLength.Store(uint32(len(t.Leechers)))

	if err = t.Group.Load(version, reader); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &id); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &snatched); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &status); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &lastAction); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &upMultiplier); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &downMultiplier); err != nil {
		return err
	}

	t.ID.Store(id)
	t.Snatched.Store(uint32(snatched))
	t.Status.Store(uint32(status))
	t.LastAction.Store(lastAction)
	t.UpMultiplier.Store(math.Float64bits(upMultiplier))
	t.DownMultiplier.Store(math.Float64bits(downMultiplier))

	return nil
}

func (t *Torrent) Append(preAllocatedBuffer []byte) (buf []byte) {
	buf = preAllocatedBuffer

	func() {
		t.PeerLock()
		defer t.PeerUnlock()

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
	}()

	buf = t.Group.Append(buf)

	buf = binary.LittleEndian.AppendUint32(buf, t.ID.Load())
	buf = binary.LittleEndian.AppendUint16(buf, uint16(t.Snatched.Load()))
	buf = append(buf, uint8(t.Status.Load()))
	buf = binary.LittleEndian.AppendUint64(buf, uint64(t.LastAction.Load()))
	buf = binary.LittleEndian.AppendUint64(buf, t.UpMultiplier.Load())
	buf = binary.LittleEndian.AppendUint64(buf, t.DownMultiplier.Load())

	return buf
}

var encodeJSONTorrentMap = make(map[string]any)
var encodeJSONTorrentGroupMap = make(map[string]any)

// MarshalJSON Due to using atomics, JSON will not marshal values within them.
// This is only safe to call from a single thread at once
func (t *Torrent) MarshalJSON() (buf []byte, err error) {
	encodeJSONTorrentMap["ID"] = t.ID.Load()
	encodeJSONTorrentMap["Seeders"] = t.Seeders
	encodeJSONTorrentMap["Leechers"] = t.Leechers

	var torrentTypeBuf [8]byte

	binary.LittleEndian.PutUint64(torrentTypeBuf[:], t.Group.TorrentType.Load())

	i := 0

	for ; i < len(torrentTypeBuf); i++ {
		if torrentTypeBuf[i] == 0 {
			break
		}
	}

	encodeJSONTorrentGroupMap["TorrentType"] = string(torrentTypeBuf[:i])
	encodeJSONTorrentGroupMap["GroupID"] = t.Group.GroupID.Load()
	encodeJSONTorrentMap["Group"] = encodeJSONTorrentGroupMap
	encodeJSONTorrentMap["Snatched"] = uint16(t.Snatched.Load())
	encodeJSONTorrentMap["Status"] = uint8(t.Status.Load())
	encodeJSONTorrentMap["LastAction"] = t.LastAction.Load()
	encodeJSONTorrentMap["UpMultiplier"] = math.Float64frombits(t.UpMultiplier.Load())
	encodeJSONTorrentMap["DownMultiplier"] = math.Float64frombits(t.UpMultiplier.Load())

	return json.Marshal(encodeJSONTorrentMap)
}

type decodeJSONTorrent struct {
	Seeders  map[PeerKey]*Peer
	Leechers map[PeerKey]*Peer

	Group struct {
		TorrentType string
		GroupID     uint32
	}
	ID       uint32
	Snatched uint16

	Status         uint8
	LastAction     int64
	UpMultiplier   float64
	DownMultiplier float64
}

// UnmarshalJSON Due to using atomics, JSON will not marshal values within them.
// This is only safe to call from a single thread at once
func (t *Torrent) UnmarshalJSON(buf []byte) (err error) {
	var torrentJSON decodeJSONTorrent
	if err = json.Unmarshal(buf, &torrentJSON); err != nil {
		return err
	}

	t.Seeders = torrentJSON.Seeders
	t.Leechers = torrentJSON.Leechers
	t.SeedersLength.Store(uint32(len(t.Seeders)))
	t.LeechersLength.Store(uint32(len(t.Leechers)))

	torrentType, err := TorrentTypeFromString(torrentJSON.Group.TorrentType)
	if err != nil {
		return err
	}

	t.Group.TorrentType.Store(torrentType)
	t.Group.GroupID.Store(torrentJSON.Group.GroupID)
	t.Snatched.Store(uint32(torrentJSON.Snatched))
	t.Status.Store(uint32(torrentJSON.Status))
	t.LastAction.Store(torrentJSON.LastAction)
	t.UpMultiplier.Store(math.Float64bits(torrentJSON.UpMultiplier))
	t.DownMultiplier.Store(math.Float64bits(torrentJSON.DownMultiplier))

	return nil
}

type TorrentGroupFreeleech struct {
	UpMultiplier   float64
	DownMultiplier float64
}

type TorrentGroupKey [8 + 4]byte

func MustTorrentGroupKeyFromString(torrentType string, groupID uint32) TorrentGroupKey {
	k, err := TorrentGroupKeyFromString(torrentType, groupID)
	if err != nil {
		panic(err)
	}

	return k
}

func TorrentGroupKeyFromString(torrentType string, groupID uint32) (k TorrentGroupKey, err error) {
	t, err := TorrentTypeFromString(torrentType)
	if err != nil {
		return TorrentGroupKey{}, err
	}

	binary.LittleEndian.PutUint64(k[:], t)
	binary.LittleEndian.PutUint32(k[8:], groupID)

	return k, nil
}

func MustTorrentTypeFromString(torrentType string) uint64 {
	t, err := TorrentTypeFromString(torrentType)
	if err != nil {
		panic(err)
	}

	return t
}

func TorrentTypeFromString(torrentType string) (t uint64, err error) {
	if len(torrentType) > 8 {
		return 0, ErrTorrentTypeTooLong
	}

	var buf [8]byte

	copy(buf[:], torrentType)

	return binary.LittleEndian.Uint64(buf[:]), nil
}

type TorrentGroup struct {
	TorrentType atomic.Uint64
	GroupID     atomic.Uint32
}

func (g *TorrentGroup) Key() (k TorrentGroupKey) {
	binary.LittleEndian.PutUint64(k[:], g.TorrentType.Load())
	binary.LittleEndian.PutUint32(k[8:], g.GroupID.Load())

	return k
}

var ErrTorrentTypeTooLong = errors.New("torrent type too long, maximum 8 bytes")

func (g *TorrentGroup) Load(version uint64, reader readerAndByteReader) (err error) {
	var (
		torrentType uint64
		groupID     uint32
	)

	if version <= 2 {
		var varIntLen uint64

		if varIntLen, err = binary.ReadUvarint(reader); err != nil {
			return err
		}

		if varIntLen > 8 {
			return ErrTorrentTypeTooLong
		}

		buf := make([]byte, 8)
		if _, err = io.ReadFull(reader, buf[:varIntLen]); err != nil {
			return err
		}

		torrentType = binary.LittleEndian.Uint64(buf)
	} else {
		if err = binary.Read(reader, binary.LittleEndian, &torrentType); err != nil {
			return err
		}
	}

	if err = binary.Read(reader, binary.LittleEndian, &groupID); err != nil {
		return err
	}

	g.TorrentType.Store(torrentType)
	g.GroupID.Store(groupID)

	return nil
}

func (g *TorrentGroup) Append(preAllocatedBuffer []byte) (buf []byte) {
	buf = preAllocatedBuffer
	buf = binary.LittleEndian.AppendUint64(buf, g.TorrentType.Load())

	return binary.LittleEndian.AppendUint32(buf, g.GroupID.Load())
}

// TorrentCacheFile holds filename used by serializer for this type
var TorrentCacheFile = "torrent-cache"

// TorrentCacheVersion Used to distinguish old versions on the on-disk cache.
// Bump when fields are altered on Torrent, Peer or TorrentGroup structs
const TorrentCacheVersion = 3

var TorrentTestCompareOptions = []cmp.Option{
	cmp.AllowUnexported(atomic.Uint32{}),
	cmp.AllowUnexported(atomic.Uint64{}),
	cmp.AllowUnexported(atomic.Int64{}),
	cmp.AllowUnexported(atomic.Bool{}),
	cmpopts.IgnoreFields(Torrent{}, "peerLock"),
}

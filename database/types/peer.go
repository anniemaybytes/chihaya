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
)

// PeerID Sent in tracker requests with client information
// https://www.bittorrent.org/beps/bep_0020.html
type PeerID [20]byte

// PeerKey Composed of an uint32 ID and a PeerID
type PeerKey [4 + 20]byte

func NewPeerKey(id uint32, peerID PeerID) (k PeerKey) {
	binary.LittleEndian.PutUint32(k[:], id)
	copy(k[4:], peerID[:])

	return k
}

//goland:noinspection GoMixedReceiverTypes
func (k PeerKey) ID() uint32 {
	return binary.LittleEndian.Uint32(k[:])
}

//goland:noinspection GoMixedReceiverTypes
func (k PeerKey) PeerID() (id PeerID) {
	copy(id[:], k[4:])

	return id
}

//goland:noinspection GoMixedReceiverTypes
func (k PeerKey) MarshalText() ([]byte, error) {
	var buf [(4 + 20) * 2]byte

	hex.Encode(buf[:], k[:])

	return buf[:], nil
}

//goland:noinspection GoMixedReceiverTypes
func (k *PeerKey) UnmarshalText(b []byte) error {
	if len(b) != (4+20)*2 {
		return errWrongPeerKeySize
	}

	if _, err := hex.Decode(k[:], b[:]); err != nil {
		return err
	}

	return nil
}

var errWrongPeerKeySize = errors.New("wrong peer key size")
var errWrongPeerIDSize = errors.New("wrong peer id size")
var errNilPeerID = errors.New("nil peer id")

func PeerIDFromRawString(buf string) (id PeerID) {
	if len(buf) != 20 {
		return
	}

	copy(id[:], buf)

	return id
}

//goland:noinspection GoMixedReceiverTypes
func (id *PeerID) Scan(src any) error {
	if src == nil {
		return nil
	} else if buf, ok := src.([]byte); ok {
		if len(buf) == 0 {
			return errNilPeerID
		}
		if len(buf) != 20 {
			return errWrongPeerIDSize
		}
		copy((*id)[:], buf)

		return nil
	}

	return errInvalidType
}

//goland:noinspection GoMixedReceiverTypes
func (id *PeerID) Value() (driver.Value, error) {
	return (*id)[:], nil
}

//goland:noinspection GoMixedReceiverTypes
func (id PeerID) MarshalText() ([]byte, error) {
	return id[:], nil
}

//goland:noinspection GoMixedReceiverTypes
func (id *PeerID) UnmarshalText(b []byte) error {
	if len(b) != 20 {
		return errWrongPeerIDSize
	}

	copy(id[:], b)

	return nil
}

type Peer struct {
	ID PeerID

	IPAddr string
	Addr   [6]byte
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

func (p *Peer) Load(reader readerAndByteReader) (err error) {
	if _, err = io.ReadFull(reader, p.ID[:]); err != nil {
		return err
	}

	var varIntLen uint64

	if varIntLen, err = binary.ReadUvarint(reader); err != nil {
		return err
	}

	buf := make([]byte, varIntLen)

	if _, err = io.ReadFull(reader, buf); err != nil {
		return err
	}

	p.IPAddr = string(buf)

	// Keep this read to maintain binary compatibility
	if _, err = binary.ReadUvarint(reader); err != nil {
		return err
	}

	if _, err = io.ReadFull(reader, p.Addr[:]); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.IP); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.Port); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.Uploaded); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.Downloaded); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.Left); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.StartTime); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.LastAnnounce); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.TorrentID); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.UserID); err != nil {
		return err
	}

	if err = binary.Read(reader, binary.LittleEndian, &p.ClientID); err != nil {
		return err
	}

	return binary.Read(reader, binary.LittleEndian, &p.Seeding)
}

func (p *Peer) Append(preAllocatedBuffer []byte) (buf []byte) {
	buf = preAllocatedBuffer
	buf = append(buf, p.ID[:]...)

	buf = binary.AppendUvarint(buf, uint64(len(p.IPAddr)))
	buf = append(buf, p.IPAddr[:]...)

	buf = binary.AppendUvarint(buf, uint64(len(p.Addr)))
	buf = append(buf, p.Addr[:]...)

	buf = binary.LittleEndian.AppendUint32(buf, p.IP)
	buf = binary.LittleEndian.AppendUint16(buf, p.Port)
	buf = binary.LittleEndian.AppendUint64(buf, p.Uploaded)
	buf = binary.LittleEndian.AppendUint64(buf, p.Downloaded)
	buf = binary.LittleEndian.AppendUint64(buf, p.Left)
	buf = binary.LittleEndian.AppendUint64(buf, uint64(p.StartTime))
	buf = binary.LittleEndian.AppendUint64(buf, uint64(p.LastAnnounce))
	buf = binary.LittleEndian.AppendUint32(buf, p.TorrentID)
	buf = binary.LittleEndian.AppendUint32(buf, p.UserID)
	buf = binary.LittleEndian.AppendUint16(buf, p.ClientID)

	if p.Seeding {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	return buf
}

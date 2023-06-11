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
	"net"
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
var errWrongPeerAddressSize = errors.New("wrong peer address size")
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

const PeerAddressSize = 4 + 2

type PeerAddress [PeerAddressSize]byte

func NewPeerAddressFromIPPort(ip net.IP, port uint16) PeerAddress {
	var a PeerAddress

	copy(a[:], ip)
	binary.BigEndian.PutUint16(a[4:], port)

	return a
}

//goland:noinspection GoMixedReceiverTypes
func (a PeerAddress) IP() net.IP {
	return a[:4]
}

//goland:noinspection GoMixedReceiverTypes
func (a PeerAddress) IPNumeric() uint32 {
	return binary.BigEndian.Uint32(a[:])
}

//goland:noinspection GoMixedReceiverTypes
func (a PeerAddress) IPString() string {
	return a.IP().String()
}

//goland:noinspection GoMixedReceiverTypes
func (a PeerAddress) Port() uint16 {
	return binary.BigEndian.Uint16(a[4:])
}

//goland:noinspection GoMixedReceiverTypes
func (a PeerAddress) MarshalText() ([]byte, error) {
	var buf [PeerAddressSize * 2]byte

	hex.Encode(buf[:], a[:])

	return buf[:], nil
}

//goland:noinspection GoMixedReceiverTypes
func (a *PeerAddress) UnmarshalText(b []byte) error {
	if len(b) != PeerAddressSize*2 {
		return errWrongPeerAddressSize
	}

	if _, err := hex.Decode(a[:], b[:]); err != nil {
		return err
	}

	return nil
}

// Peer
// Theoretical min layout size: 6 + 8 + 8 + 8 + 8 + 8 + 8 + 4 + 4 + 6 + 2 + 1 = 71 bytes
// Current layout size go1.20.4: 80 bytes via unsafe.Sizeof(Peer{})
type Peer struct {
	Addr PeerAddress

	Uploaded   uint64
	Downloaded uint64
	Left       uint64

	StartTime    int64 // unix time
	LastAnnounce int64

	TorrentID uint32
	UserID    uint32

	// ID placed here so in-memory layout is smaller
	ID PeerID

	ClientID uint16

	Seeding bool
}

var errInvalidAddrLength = errors.New("invalid Addr length")

func (p *Peer) Load(version uint64, reader readerAndByteReader) (err error) {
	if _, err = io.ReadFull(reader, p.ID[:]); err != nil {
		return err
	}

	if version == 1 {
		// Read IPAddr string
		var varIntLen uint64

		if varIntLen, err = binary.ReadUvarint(reader); err != nil {
			return err
		}

		buf := make([]byte, varIntLen)

		if _, err = io.ReadFull(reader, buf); err != nil {
			return err
		}

		// Read length of Addr
		if varIntLen, err = binary.ReadUvarint(reader); err != nil {
			return err
		}

		if int(varIntLen) != len(p.Addr) {
			return errInvalidAddrLength
		}

		if _, err = io.ReadFull(reader, p.Addr[:]); err != nil {
			return err
		}

		var (
			ip   uint32
			port uint16
		)

		if err = binary.Read(reader, binary.LittleEndian, &ip); err != nil {
			return err
		}

		if err = binary.Read(reader, binary.LittleEndian, &port); err != nil {
			return err
		}
	} else {
		if _, err = io.ReadFull(reader, p.Addr[:]); err != nil {
			return err
		}
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
	buf = append(buf, p.Addr[:]...)
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

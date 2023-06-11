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
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

type readerAndByteReader interface {
	io.Reader
	io.ByteReader
}

func WriteSerializeHeader(writer io.Writer, n int, version uint64) (err error) {
	var varIntBuf [binary.MaxVarintLen64]byte

	if _, err = writer.Write(varIntBuf[:binary.PutUvarint(varIntBuf[:], version)]); err != nil {
		return err
	}

	if _, err = writer.Write(varIntBuf[:binary.PutUvarint(varIntBuf[:], uint64(n))]); err != nil {
		return err
	}

	return nil
}

var errUnsupportedVersion = errors.New("unsupported version")

func LoadSerializeHeader(reader readerAndByteReader, maxSupportedVersion uint64) (n int, version uint64, err error) {
	var records uint64

	if version, err = binary.ReadUvarint(reader); err != nil {
		return 0, 0, err
	}

	if version == 0 || version > maxSupportedVersion {
		return 0, version, errUnsupportedVersion
	}

	if records, err = binary.ReadUvarint(reader); err != nil {
		return 0, version, err
	}

	return int(records), version, nil
}

func WriteTorrents(w io.Writer, torrents map[TorrentHash]*Torrent) error {
	writer := bufio.NewWriterSize(w, 1024*64)
	defer func(writer *bufio.Writer) {
		_ = writer.Flush()
	}(writer)

	if err := WriteSerializeHeader(writer, len(torrents), TorrentCacheVersion); err != nil {
		return err
	}

	preAllocatedBuffer := make([]byte, 0, 4096)

	for k, v := range torrents {
		buf := preAllocatedBuffer[:0]
		buf = append(buf, k[:]...)
		buf = v.Append(buf)

		if _, err := writer.Write(buf); err != nil {
			return err
		}

		preAllocatedBuffer = buf
	}

	return nil
}

func LoadTorrents(r io.Reader, torrents map[TorrentHash]*Torrent) error {
	reader := bufio.NewReader(r)

	n, version, err := LoadSerializeHeader(reader, TorrentCacheVersion)

	if err != nil {
		return err
	}

	var k TorrentHash

	for i := 0; i < n; i++ {
		if _, err := io.ReadFull(reader, k[:]); err != nil {
			return err
		}

		t := &Torrent{}

		if err := t.Load(version, reader); err != nil {
			return err
		}

		torrents[k] = t
	}

	return nil
}

func WriteUsers(w io.Writer, users map[string]*User) error {
	writer := bufio.NewWriterSize(w, 1024*64)
	defer func(writer *bufio.Writer) {
		_ = writer.Flush()
	}(writer)

	if err := WriteSerializeHeader(writer, len(users), UserCacheVersion); err != nil {
		return err
	}

	preAllocatedBuffer := make([]byte, 0, 4096)

	for k, v := range users {
		buf := preAllocatedBuffer[:0]
		buf = binary.AppendUvarint(buf, uint64(len(k)))
		buf = append(buf, k[:]...)
		buf = v.Append(buf)

		if _, err := writer.Write(buf); err != nil {
			return err
		}

		preAllocatedBuffer = buf
	}

	return nil
}

func LoadUsers(r io.Reader, users map[string]*User) error {
	reader := bufio.NewReader(r)

	n, version, err := LoadSerializeHeader(reader, UserCacheVersion)

	if err != nil {
		return err
	}

	var varIntLen uint64

	for i := 0; i < n; i++ {
		if varIntLen, err = binary.ReadUvarint(reader); err != nil {
			return err
		}

		buf := make([]byte, varIntLen)

		if _, err = io.ReadFull(reader, buf); err != nil {
			return err
		}

		u := &User{}
		if err := u.Load(version, reader); err != nil {
			return err
		}

		users[string(buf)] = u
	}

	return nil
}

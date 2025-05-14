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

package params

import (
	"bytes"
	"strconv"

	cdb "chihaya/database/types"

	"github.com/valyala/fasthttp"
)

type QueryParam struct {
	Params struct {
		Uploaded   uint64
		Downloaded uint64
		Left       uint64

		Port    uint16
		NumWant uint16

		PeerID string
		IPv4   string
		IP     string
		Event  string

		testGarbageUnescape string // for testing purposes

		Compact  bool
		NoPeerID bool

		InfoHashes []cdb.TorrentHash
	}

	Exists struct {
		Uploaded   bool
		Downloaded bool
		Left       bool

		Port    bool
		NumWant bool

		PeerID bool
		IP     bool
		Event  bool

		testGarbageUnescape bool // for testing purposes

		Compact  bool
		NoPeerID bool

		InfoHashes bool
	}
}

var uploadedKey = []byte("uploaded")
var downloadedKey = []byte("downloaded")
var leftKey = []byte("left")

var portKey = []byte("port")
var numWant = []byte("numwant")

var peerIDKey = []byte("peer_id")
var ipKey = []byte("ip")
var eventKey = []byte("event")

var testGarbageUnescapeKey = []byte("!@#") // for testing purposes

var infoHashKey = []byte("info_hash")

var compactKey = []byte("compact")
var noPeerIDKey = []byte("no_peer_id")

func ParseQuery(queryArgs *fasthttp.Args) (qp QueryParam, err error) {
	queryArgs.VisitAll(func(key, value []byte) {
		if err != nil {
			return
		}

		key = bytes.ToLower(key)

		switch true {
		case bytes.Equal(key, uploadedKey):
			n, errz := strconv.ParseUint(string(value), 10, 64)
			if errz != nil {
				err = errz
				return
			}

			qp.Params.Uploaded = n
			qp.Exists.Uploaded = true
		case bytes.Equal(key, downloadedKey):
			n, errz := strconv.ParseUint(string(value), 10, 64)
			if errz != nil {
				err = errz
				return
			}

			qp.Params.Downloaded = n
			qp.Exists.Downloaded = true
		case bytes.Equal(key, leftKey):
			n, errz := strconv.ParseUint(string(value), 10, 64)
			if errz != nil {
				err = errz
				return
			}

			qp.Params.Left = n
			qp.Exists.Left = true
		case bytes.Equal(key, portKey):
			n, errz := strconv.ParseUint(string(value), 10, 16)
			if errz != nil {
				err = errz
				return
			}

			qp.Params.Port = uint16(n)
			qp.Exists.Port = true
		case bytes.Equal(key, numWant):
			n, errz := strconv.ParseUint(string(value), 10, 16)
			if errz != nil {
				err = errz
				return
			}

			qp.Params.NumWant = uint16(n)
			qp.Exists.NumWant = true
		case bytes.Equal(key, peerIDKey):
			qp.Params.PeerID = string(value)
			qp.Exists.PeerID = true
		case bytes.Equal(key, ipKey):
			qp.Params.IP = string(value)
			qp.Exists.IP = true
		case bytes.Equal(key, eventKey):
			qp.Params.Event = string(value)
			qp.Exists.Event = true
		case bytes.Equal(key, testGarbageUnescapeKey): // for testing purposes
			qp.Params.testGarbageUnescape = string(value)
			qp.Exists.testGarbageUnescape = true
		case bytes.Equal(key, infoHashKey):
			if len(value) == cdb.TorrentHashSize {
				qp.Params.InfoHashes = append(qp.Params.InfoHashes, cdb.TorrentHashFromBytes(value))
				qp.Exists.InfoHashes = true
			}
		case bytes.Equal(key, compactKey):
			qp.Params.Compact = bytes.Equal(value, []byte{'1'})
			qp.Exists.Compact = true
		case bytes.Equal(key, noPeerIDKey):
			qp.Params.NoPeerID = bytes.Equal(value, []byte{'1'})
			qp.Exists.NoPeerID = true
		}
	})

	return qp, err
}

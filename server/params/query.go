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

// Package params is based on https://github.com/chihaya/chihaya/blob/e6e7269/bittorrent/params.go
package params

import (
	cdb "chihaya/database/types"
	"net/url"
	"strconv"
	"strings"
)

type QueryParam struct {
	query      string
	params     map[string]string
	infoHashes []cdb.TorrentHash
}

func ParseQuery(query string) (qp *QueryParam, err error) {
	qp = &QueryParam{
		query:      query,
		infoHashes: nil,
		params:     make(map[string]string),
	}

	for query != "" {
		key := query
		if i := strings.Index(key, "&"); i >= 0 {
			key, query = key[:i], key[i+1:]
		} else {
			query = ""
		}

		if key == "" {
			continue
		}

		value := ""
		if i := strings.Index(key, "="); i >= 0 {
			key, value = key[:i], key[i+1:]
		}

		key, err = url.QueryUnescape(key)
		if err != nil {
			panic(err)
		}

		value, err = url.QueryUnescape(value)
		if err != nil {
			panic(err)
		}

		if key == "info_hash" {
			if len(value) == cdb.TorrentHashSize {
				qp.infoHashes = append(qp.infoHashes, cdb.TorrentHashFromBytes([]byte(value)))
			}
		} else {
			qp.params[strings.ToLower(key)] = value
		}
	}

	return qp, nil
}

func (qp *QueryParam) getUint(which string, bitSize int) (ret uint64, exists bool) {
	str, exists := qp.params[which]
	if exists {
		var err error

		ret, err = strconv.ParseUint(str, 10, bitSize)
		if err != nil {
			exists = false
		}
	}

	return
}

func (qp *QueryParam) Get(which string) (ret string, exists bool) {
	ret, exists = qp.params[which]
	return
}

func (qp *QueryParam) GetUint64(which string) (ret uint64, exists bool) {
	return qp.getUint(which, 64)
}

func (qp *QueryParam) GetUint16(which string) (ret uint16, exists bool) {
	tmp, exists := qp.getUint(which, 16)
	ret = uint16(tmp)

	return
}

func (qp *QueryParam) InfoHashes() []cdb.TorrentHash {
	return qp.infoHashes
}

func (qp *QueryParam) RawQuery() string {
	return qp.query
}

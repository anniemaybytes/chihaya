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

// URL.Query() is rather slow, so I rewrote it
// Since the only parameter that can have multiple values is info_hash for scrapes, handle this specifically
package params

import (
	"net/url"
	"strconv"
)

type QueryParam struct {
	params     map[string]string
	infoHashes []string
}

func (qp *QueryParam) getUint(which string, bitSize int) (ret uint64, exists bool) {
	str, exists := qp.params[which]
	if exists {
		var err error

		exists = false

		ret, err = strconv.ParseUint(str, 10, bitSize)
		if err == nil {
			exists = true
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

func (qp *QueryParam) InfoHashes() []string {
	return qp.infoHashes
}

func ParseQuery(query string) (ret *QueryParam, err error) {
	ret = &QueryParam{make(map[string]string), nil}
	queryLen := len(query)

	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfoHash    string
	)

	onKey := true
	hasInfoHash := false

	for i := 0; i < queryLen; i++ {
		separator := query[i] == '&' || query[i] == ';' // ';' is a valid separator as per W3C spec
		if separator || i == queryLen-1 {
			if onKey {
				keyStart = i + 1
				continue
			}

			if i == queryLen-1 && !separator {
				if query[i] == '=' {
					continue
				}

				valEnd = i
			}

			keyStr, errz := url.QueryUnescape(query[keyStart : keyEnd+1])
			if errz != nil {
				err = errz
				return
			}

			valStr, errz := url.QueryUnescape(query[valStart : valEnd+1])
			if errz != nil {
				err = errz
				return
			}

			ret.params[keyStr] = valStr

			if keyStr == "info_hash" {
				if hasInfoHash {
					// There is more than one info_hash
					if ret.infoHashes == nil {
						ret.infoHashes = []string{firstInfoHash}
					}

					ret.infoHashes = append(ret.infoHashes, valStr)
				} else {
					firstInfoHash = valStr
					hasInfoHash = true
				}
			}

			onKey = true
			keyStart = i + 1
		} else if query[i] == '=' {
			onKey = false
			valStart = i + 1
		} else if onKey {
			keyEnd = i
		} else {
			valEnd = i
		}
	}

	return
}

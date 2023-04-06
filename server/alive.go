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

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"chihaya/log"
)

func alive(buf *bytes.Buffer) int {
	type response struct {
		Now    int64 `json:"now"`
		Uptime int64 `json:"uptime"`
	}

	res, err := json.Marshal(response{time.Now().UnixMilli(), time.Since(handler.startTime).Milliseconds()})
	if err != nil {
		log.Error.Print("Failed to marshal JSON alive response: ", err)
		return http.StatusInternalServerError
	}

	buf.Write(res)

	return http.StatusOK
}

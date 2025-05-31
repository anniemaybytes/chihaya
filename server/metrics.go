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

	"chihaya/collector"
	"chihaya/database"

	vm "github.com/VictoriaMetrics/metrics"
	"github.com/valyala/fasthttp"
)

func metrics(_ *fasthttp.RequestCtx, db *database.Database, buf *bytes.Buffer) int {
	collector.UpdateUptime(handler.startTime)
	collector.UpdatePeers(func() (c int) {
		for _, t := range *db.Torrents.Load() {
			c += int(t.LeechersLength.Load()) + int(t.SeedersLength.Load())
		}

		return
	}())

	vm.WritePrometheus(buf, true)

	return fasthttp.StatusOK
}

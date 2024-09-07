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
	"time"

	"chihaya/collector"
	"chihaya/database"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/valyala/fasthttp"
)

func metrics(_ *fasthttp.RequestCtx, db *database.Database, buf *bytes.Buffer) int {
	dbUsers := *db.Users.Load()
	dbTorrents := *db.Torrents.Load()

	peers := 0
	for _, t := range dbTorrents {
		peers += int(t.LeechersLength.Load()) + int(t.SeedersLength.Load())
	}

	collector.UpdateUptime(time.Since(handler.startTime).Seconds())
	collector.UpdateUsers(len(dbUsers))
	collector.UpdateTorrents(len(dbTorrents))
	collector.UpdateClients(len(*db.Clients.Load()))
	collector.UpdateHitAndRuns(len(*db.HitAndRuns.Load()))
	collector.UpdatePeers(peers)
	collector.UpdateRequests(handler.requests.Load())
	collector.UpdateThroughput(handler.throughput)

	mfs, _ := prometheus.DefaultGatherer.Gather()
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(buf, mf); err != nil {
			panic(err)
		}
	}

	return fasthttp.StatusOK
}

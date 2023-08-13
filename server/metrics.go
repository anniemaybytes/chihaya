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
	cdb "chihaya/database/types"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	"chihaya/database"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/valyala/fasthttp"
)

var bearerPrefix = []byte("Bearer ")

func metrics(ctx *fasthttp.RequestCtx, _ *cdb.User, db *database.Database, buf *bytes.Buffer) int {
	dbUsers := *db.Users.Load()
	dbTorrents := *db.Torrents.Load()

	peers := 0

	for _, t := range dbTorrents {
		peers += int(t.LeechersLength.Load()) + int(t.SeedersLength.Load())
	}

	collectors.UpdateUptime(time.Since(handler.startTime).Seconds())
	collectors.UpdateUsers(len(dbUsers))
	collectors.UpdateTorrents(len(dbTorrents))
	collectors.UpdateClients(len(*db.Clients.Load()))
	collectors.UpdateHitAndRuns(len(*db.HitAndRuns.Load()))
	collectors.UpdatePeers(peers)
	collectors.UpdateRequests(handler.requests.Load())
	collectors.UpdateThroughput(handler.throughput)

	mfs, _ := handler.normalRegisterer.(prometheus.Gatherer).Gather()
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(buf, mf); err != nil {
			panic(err)
		}
	}

	authString := ctx.Request.Header.PeekBytes([]byte("Authorization"))
	if len(authString) > len(bearerPrefix) && bytes.Equal(authString[:len(bearerPrefix)], bearerPrefix) {
		adminToken, exists := config.Section("http").Get("admin_token", "")
		if exists && bytes.Equal(authString[len(bearerPrefix):], []byte(adminToken)) {
			mfs, _ := prometheus.DefaultGatherer.Gather()

			for _, mf := range mfs {
				if _, err := expfmt.MetricFamilyToText(buf, mf); err != nil {
					panic(err)
				}
			}
		}
	}

	return fasthttp.StatusOK
}

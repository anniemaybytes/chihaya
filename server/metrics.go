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
	"context"
	"net/http"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	"chihaya/database"
	"chihaya/log"
	"chihaya/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

var bearerPrefix = "Bearer "

func metrics(ctx context.Context, auth string, db *database.Database, buf *bytes.Buffer) int {
	if !util.TryTakeSemaphore(ctx, db.UsersSemaphore) {
		return http.StatusRequestTimeout
	}
	defer util.ReturnSemaphore(db.UsersSemaphore)

	if !util.TryTakeSemaphore(ctx, db.TorrentsSemaphore) {
		return http.StatusRequestTimeout
	}
	defer util.ReturnSemaphore(db.TorrentsSemaphore)

	peers := 0
	for _, t := range db.Torrents {
		peers += len(t.Leechers) + len(t.Seeders)
	}

	// Early exit before response write
	select {
	case <-ctx.Done():
		return http.StatusRequestTimeout
	default:
	}

	collectors.UpdateUptime(time.Since(handler.startTime).Seconds())
	collectors.UpdateUsers(len(db.Users))
	collectors.UpdateTorrents(len(db.Torrents))
	collectors.UpdateClients(len(*db.Clients.Load()))
	collectors.UpdateHitAndRuns(len(*db.HitAndRuns.Load()))
	collectors.UpdatePeers(peers)
	collectors.UpdateRequests(handler.requests.Load())
	collectors.UpdateThroughput(handler.throughput)

	mfs, _ := handler.normalRegisterer.(prometheus.Gatherer).Gather()
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(buf, mf); err != nil {
			log.Panic.Printf("Error in converting metrics to text: %v", err)
			panic(err)
		}
	}

	n := len(bearerPrefix)
	if len(auth) > n && auth[:n] == bearerPrefix {
		adminToken, exists := config.Section("http").Get("admin_token", "")
		if exists && auth[n:] == adminToken {
			mfs, _ := prometheus.DefaultGatherer.Gather()

			for _, mf := range mfs {
				if _, err := expfmt.MetricFamilyToText(buf, mf); err != nil {
					log.Panic.Printf("Error in converting metrics to text: %v", err)
					panic(err)
				}
			}
		}
	}

	return http.StatusOK
}

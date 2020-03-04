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
	"chihaya/config"
	cdb "chihaya/database"
	"log"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

var bearerPrefix = "Bearer "

func metrics(auth string, collector *NormalCollector, db *cdb.Database, buf *bytes.Buffer) {
	peers := 0

	db.UsersMutex.RLock()
	db.TorrentsMutex.RLock()

	for _, t := range db.Torrents {
		peers += len(t.Leechers) + len(t.Seeders)
	}

	collector.UpdateUptime(time.Since(handler.startTime).Seconds())
	collector.UpdateUsers(len(db.Users))
	collector.UpdateTorrents(len(db.Torrents))
	collector.UpdatePeers(peers)
	collector.UpdateRequests(atomic.LoadUint64(&handler.requests))

	db.UsersMutex.RUnlock()
	db.TorrentsMutex.RUnlock()

	mfs, _ := handler.normalRegisterer.(prometheus.Gatherer).Gather()

	for _, mf := range mfs {
		_, err := expfmt.MetricFamilyToText(buf, mf)
		if err != nil {
			log.Printf("!!! CRITICAL !!! Error in converting metrics to text")
			return
		}
	}

	n := len(bearerPrefix)
	if len(auth) > n && auth[:n] == bearerPrefix {
		if auth[n:] == config.Get("admin_token") {
			mfs, _ := prometheus.DefaultGatherer.Gather()

			for _, mf := range mfs {
				_, err := expfmt.MetricFamilyToText(buf, mf)
				if err != nil {
					log.Printf("!!! CRITICAL !!! Error in converting metrics to text")
					return
				}
			}
		}
	}
}

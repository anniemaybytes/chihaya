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

package collector

import (
	"fmt"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

var (
	uptimeMetric     = metrics.NewFloatCounter("chihaya_uptime")
	usersMetric      = metrics.NewGauge("chihaya_users", nil)
	torrentsMetric   = metrics.NewGauge("chihaya_torrents", nil)
	clientsMetric    = metrics.NewGauge("chihaya_clients", nil)
	hitAndRunsMetric = metrics.NewGauge("chihaya_hnrs", nil)
	peersMetric      = metrics.NewGauge("chihaya_peers", nil)
	requestsMetric   = metrics.NewCounter("chihaya_requests")
	throughputMetric = metrics.NewGauge("chihaya_throughput", nil)

	deadlockCountMetric   = metrics.NewCounter("chihaya_deadlock_count")
	deadlockAbortedMetric = metrics.NewCounter("chihaya_deadlock_aborted_count")
	deadlockTimeMetric    = metrics.NewFloatCounter("chihaya_deadlock_seconds_total")
	erroredRequestsMetric = metrics.NewCounter("chihaya_requests_fail")
	sqlErrorCountMetric   = metrics.NewCounter("chihaya_sql_errors_count")

	serializationTime = metrics.NewHistogram("chihaya_serialization_seconds")
	purgePeersTime    = metrics.NewHistogram("chihaya_purge_inactive_peers_seconds")
)

func UpdateUptime(startTime time.Time) {
	uptimeMetric.Set(time.Since(startTime).Seconds())
}

func UpdateUsers(count int) {
	usersMetric.Set(float64(count))
}

func UpdatePeers(count int) {
	peersMetric.Set(float64(count))
}

func UpdateTorrents(count int) {
	torrentsMetric.Set(float64(count))
}

func UpdateClients(count int) {
	clientsMetric.Set(float64(count))
}

func UpdateHitAndRuns(count int) {
	hitAndRunsMetric.Set(float64(count))
}

func IncrementRequests() {
	requestsMetric.Inc()
}

func UpdateThroughput(rpm int) {
	throughputMetric.Set(float64(rpm))
}

func IncrementDeadlockCount() {
	deadlockCountMetric.Inc()
}

func IncrementDeadlockTime(time time.Duration) {
	deadlockTimeMetric.Set(time.Seconds())
}

func IncrementDeadlockAborted() {
	deadlockAbortedMetric.Inc()
}

func IncrementErroredRequests() {
	erroredRequestsMetric.Inc()
}

func IncrementSQLErrorCount() {
	sqlErrorCountMetric.Inc()
}

func UpdateSerializationTime(v time.Duration) {
	serializationTime.Update(v.Seconds())
}

func UpdateReloadTime(source string, time time.Duration) {
	metrics.GetOrCreateHistogram(fmt.Sprintf(`chihaya_reload_seconds{source=%q}`, source)).Update(time.Seconds())
}

func UpdatePurgeInactivePeersTime(time time.Duration) {
	purgePeersTime.Update(time.Seconds())
}

func UpdateChannelFlushTime(channel string, time time.Duration) {
	metrics.GetOrCreateHistogram(fmt.Sprintf(`chihaya_flush_seconds{channel=%q}`, channel)).Update(time.Seconds())
}

func UpdateChannelFlushLen(channel string, length int) {
	metrics.GetOrCreateHistogram(fmt.Sprintf(`chihaya_channel_len{channel=%q}`, channel)).Update(float64(length))
}

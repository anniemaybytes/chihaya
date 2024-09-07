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
	"log/slog"
	"time"

	"chihaya/config"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	uptimeMetric     *prometheus.Desc
	usersMetric      *prometheus.Desc
	torrentsMetric   *prometheus.Desc
	clientsMetric    *prometheus.Desc
	hitAndRunsMetric *prometheus.Desc
	peersMetric      *prometheus.Desc
	requestsMetric   *prometheus.Desc
	throughputMetric *prometheus.Desc

	deadlockTimeMetric    *prometheus.Desc
	deadlockCountMetric   *prometheus.Desc
	deadlockAbortedMetric *prometheus.Desc
	erroredRequestsMetric *prometheus.Desc
	sqlErrorCountMetric   *prometheus.Desc

	reloadTimeSummary *prometheus.HistogramVec
	flushTimeSummary  *prometheus.HistogramVec

	purgePeersTimeHistogram    *prometheus.Histogram
	serializationTimeHistogram *prometheus.Histogram

	torrentFlushBufferHistogram         *prometheus.Histogram
	userFlushBufferHistogram            *prometheus.Histogram
	transferHistoryFlushBufferHistogram *prometheus.Histogram
	transferIpsFlushBufferHistogram     *prometheus.Histogram
	snatchFlushBufferHistogram          *prometheus.Histogram
}

var (
	users      int
	torrents   int
	clients    int
	hitAndRuns int
	peers      int
	uptime     float64
	requests   uint64
	throughput int

	torrentFlushBufferSize         int
	userFlushBufferSize            int
	transferHistoryFlushBufferSize int
	transferIpsFlushBufferSize     int
	snatchFlushBufferSize          int
)

var (
	serializationTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_serialization_seconds",
		Help:    "Histogram of the time taken to serialize database",
		Buckets: []float64{2.5, 3, 3.5, 4, 4.5, 5, 5.5, 6, 8, 10},
	})
	reloadTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chihaya_reload_seconds",
		Help:    "Histogram of the time taken to reload data from database",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"type"})
	flushTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chihaya_flush_seconds",
		Help:    "Histogram of the time taken to flush data from individual channels to database",
		Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 1.5, 2, 5},
	}, []string{"type"})
	purgePeersTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_purge_inactive_peers_seconds",
		Help:    "Histogram of the time taken to purge inactive peers from memory",
		Buckets: []float64{.01, .05, .1, .15, .25, .35, .5, .75, 1, 1.25, 1.5, 1.75, 2.5, 5},
	})

	torrentFlushBufferLength         prometheus.Histogram
	userFlushBufferLength            prometheus.Histogram
	transferHistoryFlushBufferLength prometheus.Histogram
	transferIpsFlushBufferLength     prometheus.Histogram
	snatchFlushBufferLength          prometheus.Histogram

	deadlockTime    = time.Duration(0)
	deadlockCount   = 0
	deadlockAborted = 0
	erroredRequests = 0
	sqlErrorCount   = 0
)

func init() {
	channelsConfig := config.Section("channels")
	torrentFlushBufferSize, _ = channelsConfig.GetInt("torrents", 5000)
	userFlushBufferSize, _ = channelsConfig.GetInt("users", 5000)
	transferHistoryFlushBufferSize, _ = channelsConfig.GetInt("transfer_history", 5000)
	transferIpsFlushBufferSize, _ = channelsConfig.GetInt("transfer_ips", 5000)
	snatchFlushBufferSize, _ = channelsConfig.GetInt("snatches", 25)

	torrentFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_torrents_channel_len",
		Help:    "Histogram representing torrents channel length during flush",
		Buckets: prometheus.LinearBuckets(0, float64(torrentFlushBufferSize)*0.05, 20),
	})
	userFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_users_channel_len",
		Help:    "Histogram representing users channel length during flush",
		Buckets: prometheus.LinearBuckets(0, float64(userFlushBufferSize)*0.05, 20),
	})
	transferHistoryFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_transfer_history_channel_len",
		Help:    "Histogram representing transfer_history channel length during flush",
		Buckets: prometheus.LinearBuckets(0, float64(transferHistoryFlushBufferSize)*0.05, 20),
	})
	transferIpsFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_transfer_ips_channel_len",
		Help:    "Histogram representing transfer_ips channel length during flush",
		Buckets: prometheus.LinearBuckets(0, float64(transferIpsFlushBufferSize)*0.05, 20),
	})
	snatchFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_snatches_channel_len",
		Help:    "Histogram representing snatches channel length during flush",
		Buckets: prometheus.LinearBuckets(0, float64(snatchFlushBufferSize)*0.05, 20),
	})
}

func NewCollector() *Collector {
	return &Collector{
		uptimeMetric: prometheus.NewDesc("chihaya_uptime",
			"System uptime in seconds", nil, nil),
		usersMetric: prometheus.NewDesc("chihaya_users",
			"Number of active users in database", nil, nil),
		torrentsMetric: prometheus.NewDesc("chihaya_torrents",
			"Number of torrents currently being tracked", nil, nil),
		clientsMetric: prometheus.NewDesc("chihaya_clients",
			"Number of approved clients", nil, nil),
		hitAndRunsMetric: prometheus.NewDesc("chihaya_hnrs",
			"Number of active hit and runs registered", nil, nil),
		peersMetric: prometheus.NewDesc("chihaya_peers",
			"Number of peers currently being tracked", nil, nil),
		requestsMetric: prometheus.NewDesc("chihaya_requests",
			"Number of requests received", nil, nil),
		throughputMetric: prometheus.NewDesc("chihaya_throughput",
			"Current throughput in requests per minute", nil, nil),

		deadlockCountMetric: prometheus.NewDesc("chihaya_deadlock_count",
			"Number of unique database deadlocks encountered", nil, nil),
		deadlockAbortedMetric: prometheus.NewDesc("chihaya_deadlock_aborted_count",
			"Number of times deadlock retries were exceeded", nil, nil),
		deadlockTimeMetric: prometheus.NewDesc("chihaya_deadlock_seconds_total",
			"Total time wasted awaiting to free deadlock", nil, nil),
		erroredRequestsMetric: prometheus.NewDesc("chihaya_requests_fail",
			"Number of failed requests", nil, nil),
		sqlErrorCountMetric: prometheus.NewDesc("chihaya_sql_errors_count",
			"Number of SQL errors", nil, nil),

		reloadTimeSummary: reloadTime,
		flushTimeSummary:  flushTime,

		purgePeersTimeHistogram:    &purgePeersTime,
		serializationTimeHistogram: &serializationTime,

		torrentFlushBufferHistogram:         &torrentFlushBufferLength,
		userFlushBufferHistogram:            &userFlushBufferLength,
		transferHistoryFlushBufferHistogram: &transferHistoryFlushBufferLength,
		transferIpsFlushBufferHistogram:     &transferIpsFlushBufferLength,
		snatchFlushBufferHistogram:          &snatchFlushBufferLength,
	}
}

func (collector *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.uptimeMetric
	ch <- collector.usersMetric
	ch <- collector.torrentsMetric
	ch <- collector.clientsMetric
	ch <- collector.hitAndRunsMetric
	ch <- collector.peersMetric
	ch <- collector.requestsMetric
	ch <- collector.throughputMetric
	ch <- collector.deadlockCountMetric
	ch <- collector.deadlockAbortedMetric
	ch <- collector.deadlockTimeMetric
	ch <- collector.erroredRequestsMetric
	ch <- collector.sqlErrorCountMetric

	reloadTime.Describe(ch)
	flushTime.Describe(ch)
	purgePeersTime.Describe(ch)
	serializationTime.Describe(ch)

	torrentFlushBufferLength.Describe(ch)
	userFlushBufferLength.Describe(ch)
	transferHistoryFlushBufferLength.Describe(ch)
	transferIpsFlushBufferLength.Describe(ch)
	snatchFlushBufferLength.Describe(ch)
}

func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(collector.uptimeMetric, prometheus.CounterValue, uptime)
	ch <- prometheus.MustNewConstMetric(collector.usersMetric, prometheus.GaugeValue, float64(users))
	ch <- prometheus.MustNewConstMetric(collector.torrentsMetric, prometheus.GaugeValue, float64(torrents))
	ch <- prometheus.MustNewConstMetric(collector.clientsMetric, prometheus.GaugeValue, float64(clients))
	ch <- prometheus.MustNewConstMetric(collector.hitAndRunsMetric, prometheus.GaugeValue, float64(hitAndRuns))
	ch <- prometheus.MustNewConstMetric(collector.peersMetric, prometheus.GaugeValue, float64(peers))
	ch <- prometheus.MustNewConstMetric(collector.requestsMetric, prometheus.CounterValue, float64(requests))
	ch <- prometheus.MustNewConstMetric(collector.throughputMetric, prometheus.GaugeValue, float64(throughput))
	ch <- prometheus.MustNewConstMetric(collector.deadlockCountMetric, prometheus.CounterValue, float64(deadlockCount))
	ch <- prometheus.MustNewConstMetric(collector.deadlockAbortedMetric, prometheus.CounterValue, float64(deadlockAborted))
	ch <- prometheus.MustNewConstMetric(collector.deadlockTimeMetric, prometheus.CounterValue, deadlockTime.Seconds())
	ch <- prometheus.MustNewConstMetric(collector.erroredRequestsMetric, prometheus.CounterValue, float64(erroredRequests))
	ch <- prometheus.MustNewConstMetric(collector.sqlErrorCountMetric, prometheus.CounterValue, float64(sqlErrorCount))

	reloadTime.Collect(ch)
	flushTime.Collect(ch)
	purgePeersTime.Collect(ch)
	serializationTime.Collect(ch)

	torrentFlushBufferLength.Collect(ch)
	userFlushBufferLength.Collect(ch)
	transferHistoryFlushBufferLength.Collect(ch)
	transferIpsFlushBufferLength.Collect(ch)
	snatchFlushBufferLength.Collect(ch)
}

func UpdateUptime(seconds float64) {
	uptime = seconds
}

func UpdateUsers(count int) {
	users = count
}

func UpdatePeers(count int) {
	peers = count
}

func UpdateTorrents(count int) {
	torrents = count
}

func UpdateClients(count int) {
	clients = count
}

func UpdateHitAndRuns(count int) {
	hitAndRuns = count
}

func UpdateRequests(count uint64) {
	requests = count
}

func UpdateThroughput(rpm int) {
	throughput = rpm
}

func IncrementDeadlockCount() {
	deadlockCount++
}

func IncrementDeadlockTime(time time.Duration) {
	deadlockTime += time
}

func IncrementDeadlockAborted() {
	deadlockAborted++
}

func IncrementErroredRequests() {
	erroredRequests++
}

func IncrementSQLErrorCount() {
	sqlErrorCount++
}

func UpdateSerializationTime(time time.Duration) {
	serializationTime.Observe(time.Seconds())
}

func UpdateReloadTime(source string, time time.Duration) {
	reloadTime.WithLabelValues(source).Observe(time.Seconds())
}

func UpdatePurgeInactivePeersTime(time time.Duration) {
	purgePeersTime.Observe(time.Seconds())
}

func UpdateChannelFlushTime(channel string, time time.Duration) {
	flushTime.WithLabelValues(channel).Observe(time.Seconds())
}

func UpdateChannelFlushLen(channel string, length int) {
	switch channel {
	case "torrents":
		torrentFlushBufferLength.Observe(float64(length))
	case "users":
		userFlushBufferLength.Observe(float64(length))
	case "transfer_history":
		transferHistoryFlushBufferLength.Observe(float64(length))
	case "transfer_ips":
		transferIpsFlushBufferLength.Observe(float64(length))
	case "snatches":
		snatchFlushBufferLength.Observe(float64(length))
	default:
		slog.Error("trying to update channel length for unknown type", "channel", channel)
	}
}

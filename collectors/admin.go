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

package collectors

import (
	"chihaya/config"
	"chihaya/log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type AdminCollector struct {
	deadlockTimeMetric  *prometheus.Desc
	deadlockCountMetric *prometheus.Desc

	serializationTimeSummary *prometheus.Histogram
	reloadTimeSummary        *prometheus.HistogramVec
	flushTimeSummary         *prometheus.HistogramVec

	torrentFlushBufferHistogram         *prometheus.Histogram
	userFlushBufferHistogram            *prometheus.Histogram
	transferHistoryFlushBufferHistogram *prometheus.Histogram
	transferIpsFlushBufferHistogram     *prometheus.Histogram
	snatchFlushBufferHistogram          *prometheus.Histogram
}

var (
	// Data
	serializationTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_serialization_seconds",
		Help:    "Histogram of the time taken to serialize database",
		Buckets: []float64{.25, .5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5},
	})
	reloadTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chihaya_reload_seconds",
		Help:    "Histogram of the time taken to reload data from database",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"type"})
	flushTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chihaya_flush_seconds",
		Help:    "Histogram of the time taken to flush data from channels to database",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"type"})

	torrentFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_torrents_channel_len",
		Help:    "Histogram representing channel length for torrents during flush",
		Buckets: prometheus.LinearBuckets(0, float64(config.TorrentFlushBufferSize)*0.05, int(1/0.05)),
	})
	userFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_users_channel_len",
		Help:    "Histogram representing channel length for users during flush",
		Buckets: prometheus.LinearBuckets(0, float64(config.UserFlushBufferSize)*0.05, int(1/0.05)),
	})
	transferHistoryFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_transfer_history_channel_len",
		Help:    "Histogram representing channel length for transfer history during flush",
		Buckets: prometheus.LinearBuckets(0, float64(config.TransferHistoryFlushBufferSize)*0.05, int(1/0.05)),
	})
	transferIpsFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_transfer_ips_channel_len",
		Help:    "Histogram representing channel length for transfer ips during flush",
		Buckets: prometheus.LinearBuckets(0, float64(config.TransferIpsFlushBufferSize)*0.05, int(1/0.05)),
	})
	snatchFlushBufferLength = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_snatches_channel_len",
		Help:    "Histogram representing channel length for snatches during flush",
		Buckets: prometheus.LinearBuckets(0, float64(config.SnatchFlushBufferSize)*0.05, int(1/0.05)),
	})

	deadlockTime  = time.Duration(0)
	deadlockCount = 0
)

func NewAdminCollector() *AdminCollector {
	return &AdminCollector{
		deadlockCountMetric: prometheus.NewDesc("chihaya_deadlock_count", "Number of times deadlock was encountered during flush", nil, nil),
		deadlockTimeMetric:  prometheus.NewDesc("chihaya_deadlock_seconds_total", "Total time wasted awaiting to free deadlock", nil, nil),

		torrentFlushBufferHistogram:         &torrentFlushBufferLength,
		userFlushBufferHistogram:            &userFlushBufferLength,
		transferHistoryFlushBufferHistogram: &transferHistoryFlushBufferLength,
		transferIpsFlushBufferHistogram:     &transferIpsFlushBufferLength,
		snatchFlushBufferHistogram:          &snatchFlushBufferLength,

		serializationTimeSummary: &serializationTime,
		reloadTimeSummary:        reloadTime,
		flushTimeSummary:         flushTime,
	}
}

func (collector *AdminCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.deadlockTimeMetric
	ch <- collector.deadlockCountMetric

	serializationTime.Describe(ch)
	reloadTime.Describe(ch)
	flushTime.Describe(ch)

	torrentFlushBufferLength.Describe(ch)
	userFlushBufferLength.Describe(ch)
	transferHistoryFlushBufferLength.Describe(ch)
	transferIpsFlushBufferLength.Describe(ch)
	snatchFlushBufferLength.Describe(ch)
}

func (collector *AdminCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(collector.deadlockCountMetric, prometheus.CounterValue, float64(deadlockCount))
	ch <- prometheus.MustNewConstMetric(collector.deadlockTimeMetric, prometheus.CounterValue, deadlockTime.Seconds())

	serializationTime.Collect(ch)
	reloadTime.Collect(ch)
	flushTime.Collect(ch)

	torrentFlushBufferLength.Collect(ch)
	userFlushBufferLength.Collect(ch)
	transferHistoryFlushBufferLength.Collect(ch)
	transferIpsFlushBufferLength.Collect(ch)
	snatchFlushBufferLength.Collect(ch)
}

func IncrementDeadlockCount() {
	deadlockCount++
}

func IncrementDeadlockTime(time time.Duration) {
	deadlockTime += time
}

func UpdateSerializationTime(time time.Duration) {
	serializationTime.Observe(time.Seconds())
}

func UpdateFlushTime(flushType string, time time.Duration) {
	flushTime.WithLabelValues(flushType).Observe(time.Seconds())
}

func UpdateReloadTime(reloadType string, time time.Duration) {
	reloadTime.WithLabelValues(reloadType).Observe(time.Seconds())
}

func UpdateChannelsLen(channelType string, length int) {
	switch channelType {
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
		log.Error.Printf("Trying to update channel length for unknown type %s", channelType)
	}
}

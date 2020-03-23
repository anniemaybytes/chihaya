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
	"github.com/prometheus/client_golang/prometheus"
)

type NormalCollector struct {
	uptimeMetric     *prometheus.Desc
	usersMetric      *prometheus.Desc
	torrentsMetric   *prometheus.Desc
	whitelistMetric  *prometheus.Desc
	hitAndRunsMetric *prometheus.Desc
	peersMetric      *prometheus.Desc
	requestsMetric   *prometheus.Desc
}

var ( // Data
	users      int
	torrents   int
	whitelist  int
	hitAndRuns int
	peers      int
	uptime     float64
	requests   uint64
)

func NewNormalCollector() *NormalCollector {
	return &NormalCollector{
		uptimeMetric:     prometheus.NewDesc("chihaya_uptime", "System uptime in seconds", nil, nil),
		usersMetric:      prometheus.NewDesc("chihaya_users", "Number of active users in database", nil, nil),
		torrentsMetric:   prometheus.NewDesc("chihaya_torrents", "Number of torrents currently being tracked", nil, nil),
		whitelistMetric:  prometheus.NewDesc("chihaya_whitelist", "Number of whitelist entries", nil, nil),
		hitAndRunsMetric: prometheus.NewDesc("chihaya_hnrs", "Number of active hit and runs registered", nil, nil),
		peersMetric:      prometheus.NewDesc("chihaya_peers", "Number of peers currently being tracked", nil, nil),
		requestsMetric:   prometheus.NewDesc("chihaya_requests", "Number of successful requests handled", nil, nil),
	}
}

func (collector *NormalCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.uptimeMetric
	ch <- collector.usersMetric
	ch <- collector.torrentsMetric
	ch <- collector.whitelistMetric
	ch <- collector.hitAndRunsMetric
	ch <- collector.peersMetric
	ch <- collector.requestsMetric
}

func (collector *NormalCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(collector.uptimeMetric, prometheus.CounterValue, uptime)
	ch <- prometheus.MustNewConstMetric(collector.usersMetric, prometheus.GaugeValue, float64(users))
	ch <- prometheus.MustNewConstMetric(collector.torrentsMetric, prometheus.GaugeValue, float64(torrents))
	ch <- prometheus.MustNewConstMetric(collector.whitelistMetric, prometheus.GaugeValue, float64(whitelist))
	ch <- prometheus.MustNewConstMetric(collector.hitAndRunsMetric, prometheus.GaugeValue, float64(hitAndRuns))
	ch <- prometheus.MustNewConstMetric(collector.peersMetric, prometheus.GaugeValue, float64(peers))
	ch <- prometheus.MustNewConstMetric(collector.requestsMetric, prometheus.CounterValue, float64(requests))
}

func UpdateUptime(tempUptime float64) {
	uptime = tempUptime
}

func UpdateUsers(tempUsers int) {
	users = tempUsers
}

func UpdatePeers(tempPeers int) {
	peers = tempPeers
}

func UpdateTorrents(tempTorrents int) {
	torrents = tempTorrents
}

func UpdateWhitelist(tempWhitelist int) {
	whitelist = tempWhitelist
}

func UpdateHitAndRuns(tempHitAndRuns int) {
	hitAndRuns = tempHitAndRuns
}

func UpdateRequests(tempRequests uint64) {
	requests = tempRequests
}

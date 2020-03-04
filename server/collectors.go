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
	"github.com/prometheus/client_golang/prometheus"
)

type NormalCollector struct {
	uptimeMetric   *prometheus.Desc
	usersMetric    *prometheus.Desc
	torrentsMetric *prometheus.Desc
	peersMetric    *prometheus.Desc
	requestsMetric *prometheus.Desc

	// Data
	users    int
	torrents int
	peers    int
	uptime   float64
	requests uint64
}

func NewNormalCollector() *NormalCollector {
	return &NormalCollector{
		uptimeMetric:   prometheus.NewDesc("chihaya_uptime", "System uptime", nil, nil),
		usersMetric:    prometheus.NewDesc("chihaya_users", "Number of active users in database", nil, nil),
		torrentsMetric: prometheus.NewDesc("chihaya_torrents", "Number of torrents currently being tracked", nil, nil),
		peersMetric:    prometheus.NewDesc("chihaya_peers", "Number of peers currently being tracked", nil, nil),
		requestsMetric: prometheus.NewDesc("chihaya_requests", "Number of requests handled", nil, nil),
	}
}

func (collector *NormalCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.uptimeMetric
	ch <- collector.usersMetric
	ch <- collector.torrentsMetric
	ch <- collector.peersMetric
	ch <- collector.requestsMetric
}

func (collector *NormalCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(collector.uptimeMetric, prometheus.CounterValue, collector.uptime)
	ch <- prometheus.MustNewConstMetric(collector.usersMetric, prometheus.GaugeValue, float64(collector.users))
	ch <- prometheus.MustNewConstMetric(collector.torrentsMetric, prometheus.GaugeValue, float64(collector.torrents))
	ch <- prometheus.MustNewConstMetric(collector.peersMetric, prometheus.GaugeValue, float64(collector.peers))
	ch <- prometheus.MustNewConstMetric(collector.requestsMetric, prometheus.CounterValue, float64(collector.requests))
}

func (collector *NormalCollector) UpdateUptime(uptime float64) {
	collector.uptime = uptime
}

func (collector *NormalCollector) UpdateUsers(users int) {
	collector.users = users
}

func (collector *NormalCollector) UpdatePeers(peers int) {
	collector.peers = peers
}

func (collector *NormalCollector) UpdateTorrents(torrents int) {
	collector.torrents = torrents
}

func (collector *NormalCollector) UpdateRequests(requests uint64) {
	collector.requests = requests
}

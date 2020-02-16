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

package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// Loaded from the database
var GlobalFreeleech = false

// Intervals
var (
	AnnounceInterval    = 45 * time.Minute
	MinAnnounceInterval = 30 * time.Minute

	DatabaseReloadInterval        = 45 * time.Second
	DatabaseSerializationInterval = 68 * time.Second
	PurgeInactiveInterval         = 120 * time.Second
)

// Time to sleep between flushes if the buffer is less than half full
var FlushSleepInterval = 3000 * time.Millisecond

// Initial time to wait before retrying the query when the database deadlocks (ramps linearly)
var DeadlockWaitTime = 1000 * time.Millisecond

// Maximum times to retry a deadlocked query before giving up
var MaxDeadlockRetries = 20

// Buffer sizes, see @Database.startFlushing()
var (
	TorrentFlushBufferSize         = 10000
	UserFlushBufferSize            = 10000
	TransferHistoryFlushBufferSize = 10000
	TransferIpsFlushBufferSize     = 10000
	SnatchFlushBufferSize          = 100
)

const LogFlushes = true

// Config file stuff
var once sync.Once

type ConfigMap map[string]interface{}

var config ConfigMap

func Get(s string) string {
	once.Do(readConfig)
	return config.Get(s)
}

func Section(s string) ConfigMap {
	once.Do(readConfig)
	return config.Section(s)
}

func (m ConfigMap) Get(s string) string {
	result, _ := m[s].(string)
	return result
}

func (m ConfigMap) Section(s string) ConfigMap {
	result, _ := m[s].(map[string]interface{})
	return result
}

func readConfig() {
	configFile := "config.json"
	f, err := os.Open(configFile)

	if err != nil {
		log.Fatalf("Error opening config file: %s", err)
		return
	}

	err = json.NewDecoder(f).Decode(&config)

	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
		return
	}
}

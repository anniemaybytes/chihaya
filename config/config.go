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
	"chihaya/log"
	"encoding/json"
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
	MinScrapeInterval   = 15 * time.Minute

	DatabaseReloadInterval        = 45 * time.Second
	DatabaseSerializationInterval = 68 * time.Second
	PurgeInactiveInterval         = 120 * time.Second

	// Time to sleep between flushes if the buffer is less than half full
	FlushSleepInterval = 3000 * time.Millisecond

	// Initial time to wait before retrying the query when the database deadlocks (ramps linearly)
	DeadlockWaitTime = 1000 * time.Millisecond

	// Maximum times to retry a deadlocked query before giving up
	MaxDeadlockRetries = 20
)

// Buffer sizes, see @Database.startFlushing()
var (
	TorrentFlushBufferSize         = 10000
	UserFlushBufferSize            = 10000
	TransferHistoryFlushBufferSize = 10000
	TransferIpsFlushBufferSize     = 10000
	SnatchFlushBufferSize          = 100
)

// Config file stuff
var (
	configFile = "config.json"
	config     ConfigMap
	once       sync.Once
)

type ConfigMap map[string]interface{}

func Get(s string, defaultValue string) string {
	once.Do(readConfig)
	return config.Get(s, defaultValue)
}

func GetBool(s string, defaultValue bool) bool {
	once.Do(readConfig)
	return config.GetBool(s, defaultValue)
}

func Section(s string) ConfigMap {
	once.Do(readConfig)
	return config.Section(s)
}

func (m ConfigMap) Get(s string, defaultValue string) string {
	if result, exists := m[s]; exists {
		return result.(string)
	} else {
		return defaultValue
	}
}

func (m ConfigMap) GetBool(s string, defaultValue bool) bool {
	if result, exists := m[s]; exists {
		return result.(bool)
	} else {
		return defaultValue
	}
}

func (m ConfigMap) Section(s string) ConfigMap {
	result, _ := m[s].(map[string]interface{})
	return result
}

func readConfig() {
	f, err := os.Open(configFile)

	if err != nil {
		log.Warning.Printf("Error opening config file, defaults will be used! (%s)", err)
		return
	}

	err = json.NewDecoder(f).Decode(&config)

	if err != nil {
		log.Warning.Printf("Error parsing config file, defaults will be used! (%s)", err)
		return
	}
}

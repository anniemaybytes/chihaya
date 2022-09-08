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
	"os"
	"sync"

	"chihaya/log"
)

var (
	configFile = "config.json"
	config     Map
	once       sync.Once
)

type Map map[string]interface{}

func Get(s string, defaultValue string) (string, bool) {
	once.Do(readConfig)
	return config.Get(s, defaultValue)
}

func GetBool(s string, defaultValue bool) (bool, bool) {
	once.Do(readConfig)
	return config.GetBool(s, defaultValue)
}

func GetInt(s string, defaultValue int) (int, bool) {
	once.Do(readConfig)
	return config.GetInt(s, defaultValue)
}

func Section(s string) Map {
	once.Do(readConfig)
	return config.Section(s)
}

func (m Map) Get(s string, defaultValue string) (string, bool) {
	if result, exists := m[s].(string); exists {
		return result, true
	}

	return defaultValue, false
}

func (m Map) GetInt(s string, defaultValue int) (int, bool) {
	if result, exists := m[s].(json.Number); exists {
		res, _ := result.Int64()
		return int(res), true
	}

	return defaultValue, false
}

func (m Map) GetBool(s string, defaultValue bool) (bool, bool) {
	if result, exists := m[s].(bool); exists {
		return result, true
	}

	return defaultValue, false
}

func (m Map) Section(s string) Map {
	result, _ := m[s].(map[string]interface{})
	return result
}

func readConfig() {
	f, err := os.Open(configFile)
	if err != nil {
		log.Warning.Printf("Unable to open config file, defaults will be used: %v", err)
		return
	}

	decoder := json.NewDecoder(f)
	decoder.UseNumber()

	if err = decoder.Decode(&config); err != nil {
		log.Error.Printf("Can not parse config file, defaults will be used: %v", err)
		return
	}
}

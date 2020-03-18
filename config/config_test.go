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
	"chihaya/util"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var configTest ConfigMap

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	tempPath := filepath.Join(os.TempDir(), "chihaya_config-"+util.RandStringBytes(6))

	err := os.Mkdir(tempPath, 0755)
	if err != nil {
		panic(err)
	}

	err = os.Chdir(tempPath)
	if err != nil {
		panic(err)
	}

	configFile = "test_config.json"

	f, err := os.OpenFile(configFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}

	configTest = make(ConfigMap)
	dbConfig := map[string]interface{}{
		"username": "chihaya",
		"password": "",
		"proto":    "tcp",
		"addr":     "127.0.0.1:3306",
		"database": "chihaya",
	}
	configTest["database"] = dbConfig
	configTest["addr"] = ":34000"

	err = json.NewEncoder(f).Encode(&configTest)
	if err != nil {
		panic(err)
	}

	_ = f.Close()

	os.Exit(m.Run())
}

func TestReadConfig(t *testing.T) {
	once.Do(readConfig)

	if config == nil {
		t.Fatalf("Config is nil!")
	}

	same := reflect.DeepEqual(config, configTest)
	if !same {
		t.Fatalf("Config (%v) was not same as the config that was written (%v)!", config, configTest)
	}

	t.Cleanup(cleanup)
}

func TestGet(t *testing.T) {
	got, _ := Get("addr", "")
	expected := configTest["addr"]

	if got != expected {
		t.Fatalf("Got %s whereas expected %s for \"addr\"!", got, expected)
	}
}

func TestGetDefault(t *testing.T) {
	got, _ := Get("idontexist", "iamdefault")

	if got != "iamdefault" {
		t.Fatalf("Got %s whereas expected iamdefault for \"idontexist\"!", got)
	}
}

func TestSection(t *testing.T) {
	got := Section("database")
	gotMap := make(map[string]interface{}, len(got))

	for k, v := range got {
		gotMap[k] = v
	}

	expected := configTest["database"]
	same := reflect.DeepEqual(gotMap, expected)

	if !same {
		t.Fatalf("Got (%v) whereas expected (%v) for \"database\"", gotMap, expected)
	}
}

func cleanup() {
	_ = os.Remove(configFile)
}

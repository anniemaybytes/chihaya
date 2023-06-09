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
	"reflect"
	"testing"
)

var configTest Map

func TestMain(m *testing.M) {
	tempPath, err := os.MkdirTemp(os.TempDir(), "chihaya_config-*")
	if err != nil {
		panic(err)
	}

	if err := os.Chmod(tempPath, 0755); err != nil {
		panic(err)
	}

	if err := os.Chdir(tempPath); err != nil {
		panic(err)
	}

	f, err := os.OpenFile("config.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}

	configTest = make(Map)
	dbConfig := map[string]interface{}{
		"username": "chihaya",
		"password": "",
		"proto":    "tcp",
		"addr":     "127.0.0.1:3306",
		"database": "chihaya",
	}
	configTest["database"] = dbConfig
	configTest["addr"] = ":34000"
	configTest["numwant"] = json.Number("25")
	configTest["log_flushes"] = true

	if err = json.NewEncoder(f).Encode(&configTest); err != nil {
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

	if same := reflect.DeepEqual(config, configTest); !same {
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

func TestGetBool(t *testing.T) {
	got, _ := GetBool("log_flushes", false)
	expected := configTest["log_flushes"]

	if got != expected {
		t.Fatalf("Got %v whereas expected %v for \"testbool\"!", got, expected)
	}
}

func TestGetBoolDefault(t *testing.T) {
	got, _ := GetBool("idontexist", true)

	if got != true {
		t.Fatalf("Got %v whereas expected true for \"boolnotexist\"!", got)
	}
}

func TestGetInt(t *testing.T) {
	got, _ := GetInt("numwant", 0)
	expectedNumber, _ := configTest["numwant"].(json.Number).Int64()
	expected := int(expectedNumber)

	if got != expected {
		t.Fatalf("Got %v whereas expected %v for \"testint\"!", got, expected)
	}
}

func TestGetIntDefault(t *testing.T) {
	got, _ := GetInt("idontexist", 64)

	if got != 64 {
		t.Fatalf("Got %v whereas expected 64 for \"intnotexist\"!", got)
	}
}

func TestSection(t *testing.T) {
	got := Section("database")
	gotMap := make(map[string]interface{}, len(got))

	for k, v := range got {
		gotMap[k] = v
	}

	expected := configTest["database"]
	if same := reflect.DeepEqual(gotMap, expected); !same {
		t.Fatalf("Got (%v) whereas expected (%v) for \"database\"", gotMap, expected)
	}
}

func cleanup() {
	_ = os.Remove("config.json")
}

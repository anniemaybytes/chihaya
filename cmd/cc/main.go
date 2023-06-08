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

package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	cdb "chihaya/database/types"
)

// provided at compile-time
var (
	BuildDate    = "0000-00-00T00:00:00+0000"
	BuildVersion = "development"
)

func help() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Println("  dump       umarashals gob cache files into readable JSON files")
	fmt.Println("  restore    marshals JSON files back into gob cache")
}

func main() {
	fmt.Printf("cache utility for chihaya (kuroneko), ver=%s date=%s runtime=%s\n\n",
		BuildVersion, BuildDate, runtime.Version())

	if len(os.Args) < 2 {
		help()
		return
	}

	switch os.Args[1] {
	case "dump":
		dump(make(map[string]*cdb.Torrent), cdb.TorrentCacheFile)
		dump(make(map[string]*cdb.User), cdb.UserCacheFile)

		return
	case "restore":
		restore(make(map[string]*cdb.Torrent), cdb.TorrentCacheFile)
		restore(make(map[string]*cdb.User), cdb.UserCacheFile)

		return
	default:
		help()
	}
}

func dump[cdb any](v cdb, f string) {
	fmt.Printf("Dumping data for %s, this might take a while...", f)

	gobFile, err := os.OpenFile(fmt.Sprintf("%s.gob", f), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	jsonFile, err := os.OpenFile(fmt.Sprintf("%s.json", f), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}

	if err = gob.NewDecoder(gobFile).Decode(&v); err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "\t")

	if err = encoder.Encode(v); err != nil {
		panic(err)
	}

	_ = gobFile.Close()
	_ = jsonFile.Close()

	fmt.Println("...Done!")
}

func restore[cdb any](v cdb, f string) {
	fmt.Printf("Restoring data for %s, this might take a while...", f)

	jsonFile, err := os.OpenFile(fmt.Sprintf("%s.json", f), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	gobFile, err := os.OpenFile(fmt.Sprintf("%s.gob", f), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}

	if err = json.NewDecoder(jsonFile).Decode(&v); err != nil {
		panic(err)
	}

	if err = gob.NewEncoder(gobFile).Encode(&v); err != nil {
		panic(err)
	}

	_ = jsonFile.Close()
	_ = gobFile.Close()

	fmt.Println("...Done!")
}

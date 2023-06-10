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
	"encoding/json"
	"fmt"
	"io"
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
	fmt.Println("  dump       unmarshals binary cache files into readable JSON files")
	fmt.Println("  restore    marshals JSON files back into binary cache")
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
		dump(func(reader io.Reader) (map[cdb.TorrentHash]*cdb.Torrent, error) {
			t := make(map[cdb.TorrentHash]*cdb.Torrent)
			if err := cdb.LoadTorrents(reader, t); err != nil {
				return nil, err
			}
			return t, nil
		}, cdb.TorrentCacheFile)
		dump(func(reader io.Reader) (map[string]*cdb.User, error) {
			u := make(map[string]*cdb.User)
			if err := cdb.LoadUsers(reader, u); err != nil {
				return nil, err
			}
			return u, nil
		}, cdb.UserCacheFile)

		return
	case "restore":
		restore(func(writer io.Writer, v map[cdb.TorrentHash]*cdb.Torrent) error {
			return cdb.WriteTorrents(writer, v)
		}, cdb.TorrentCacheFile)
		restore(func(writer io.Writer, v map[string]*cdb.User) error {
			return cdb.WriteUsers(writer, v)
		}, cdb.UserCacheFile)

		return
	default:
		help()
	}
}

func dump[cdb any](readFunc func(reader io.Reader) (cdb, error), f string) {
	fmt.Printf("Dumping data for %s, this might take a while...", f)

	binFile, err := os.OpenFile(fmt.Sprintf("%s.bin", f), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	jsonFile, err := os.OpenFile(fmt.Sprintf("%s.json", f), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}

	var v cdb

	if v, err = readFunc(binFile); err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "\t")

	if err = encoder.Encode(v); err != nil {
		panic(err)
	}

	_ = binFile.Close()
	_ = jsonFile.Close()

	fmt.Println("...Done!")
}

func restore[cdb any](writeFunc func(writer io.Writer, v cdb) error, f string) {
	fmt.Printf("Restoring data for %s, this might take a while...", f)

	jsonFile, err := os.OpenFile(fmt.Sprintf("%s.json", f), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	binFile, err := os.OpenFile(fmt.Sprintf("%s.bin", f), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}

	var v cdb
	if err = json.NewDecoder(jsonFile).Decode(&v); err != nil {
		panic(err)
	}

	if err = writeFunc(binFile, v); err != nil {
		panic(err)
	}

	_ = jsonFile.Close()
	_ = binFile.Close()

	fmt.Println("...Done!")
}

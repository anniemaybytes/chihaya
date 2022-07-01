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
	cdb "chihaya/database/types"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

// provided at compile-time
var (
	BuildDate    = "0000-00-00T00:00:00+0000"
	BuildVersion = "development"
)

func printHelp() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Println("  dump       umarashals gob cache files into readable JSON files")
	fmt.Println("  restore    marshals JSON files back into gob cache")
}

func main() {
	fmt.Printf("cache utility for chihaya (kuroneko), ver=%s date=%s runtime=%s\n\n",
		BuildVersion, BuildDate, runtime.Version())

	if len(os.Args) < 2 {
		printHelp()
		return
	}

	pattern := os.Args[1]
	switch pattern {
	case "dump":
		dumpCache()
		return
	case "restore":
		restoreCache()
		return
	}

	printHelp()
}

func dumpCache() {
	torrents := make(map[string]*cdb.Torrent)
	users := make(map[string]*cdb.User)

	// dump torrent data
	torrentGobFile, err := os.OpenFile(fmt.Sprintf("%s.gob", cdb.TorrentCacheFile), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Println("Dumping data for torrent cache, this might take a while...")

	if err = gob.NewDecoder(torrentGobFile).Decode(&torrents); err != nil {
		panic(err)
	}

	if err = torrentGobFile.Close(); err != nil {
		panic(err)
	}

	res, err := json.MarshalIndent(torrents, "", "\t")
	if err != nil {
		panic(err)
	}

	torrentJSONFile, err := os.OpenFile(fmt.Sprintf("%s.json", cdb.TorrentCacheFile), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	if _, err = torrentJSONFile.Write(res); err != nil {
		panic(err)
	}

	if err = torrentJSONFile.Close(); err != nil {
		panic(err)
	}

	peersLen := 0
	torrentsLen := len(torrents)

	for _, t := range torrents {
		peersLen += len(t.Leechers) + len(t.Seeders)
	}

	fmt.Printf("Done! Exported %d torrent entries with %d peers\n", torrentsLen, peersLen)

	// dump user data
	userGobFile, err := os.OpenFile(fmt.Sprintf("%s.gob", cdb.UserCacheFile), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Println("Dumping data for user cache, this might take a while...")

	if err = gob.NewDecoder(userGobFile).Decode(&users); err != nil {
		panic(err)
	}

	res, err = json.MarshalIndent(users, "", "\t")
	if err != nil {
		panic(err)
	}

	if err = userGobFile.Close(); err != nil {
		panic(err)
	}

	userJSONFile, err := os.OpenFile(fmt.Sprintf("%s.json", cdb.UserCacheFile), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	if _, err = userJSONFile.Write(res); err != nil {
		panic(err)
	}

	if err = userJSONFile.Close(); err != nil {
		panic(err)
	}

	usersLen := len(users)

	fmt.Printf("Done! Exported %d user entries\n", usersLen)
}

func restoreCache() {
	torrents := make(map[string]*cdb.Torrent)
	users := make(map[string]*cdb.User)

	// restore torrent data
	torrentJSONFile, err := os.OpenFile(fmt.Sprintf("%s.json", cdb.TorrentCacheFile), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	torrentGobFile, err := os.OpenFile(fmt.Sprintf("%s.gob", cdb.TorrentCacheFile), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Println("Restoring data for torrent cache, this might take a while...")

	if err = json.NewDecoder(torrentJSONFile).Decode(&torrents); err != nil {
		panic(err)
	}

	if err = torrentJSONFile.Close(); err != nil {
		panic(err)
	}

	if err = gob.NewEncoder(torrentGobFile).Encode(&torrents); err != nil {
		panic(err)
	}

	if err = torrentGobFile.Close(); err != nil {
		panic(err)
	}

	peersLen := 0
	torrentsLen := len(torrents)

	for _, t := range torrents {
		peersLen += len(t.Leechers) + len(t.Seeders)
	}

	fmt.Printf("Done! Imported %d torrent entries with %d peers\n", torrentsLen, peersLen)

	// restore user data
	userJSONFile, err := os.OpenFile(fmt.Sprintf("%s.json", cdb.UserCacheFile), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	userGobFile, err := os.OpenFile(fmt.Sprintf("%s.gob", cdb.UserCacheFile), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Println("Restoring data for user cache, this might take a while...")

	if err = json.NewDecoder(userJSONFile).Decode(&users); err != nil {
		panic(err)
	}

	if err = userJSONFile.Close(); err != nil {
		panic(err)
	}

	if err = gob.NewEncoder(userGobFile).Encode(&users); err != nil {
		panic(err)
	}

	if err = userGobFile.Close(); err != nil {
		panic(err)
	}

	usersLen := len(users)

	fmt.Printf("Done! Imported %d user entries\n", usersLen)
}

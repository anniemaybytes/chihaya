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
	BuildDate    = "undefined"
	BuildVersion = "development"
)

func printHelp() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Println("  dump       umarashals gob cache files into readable JSON files")
	fmt.Println("  restore    marshals JSON files back into gob cache")
}

func main() {
	fmt.Printf("cache utility for chihaya (kuroneko), build=%s date=%s runtime=%s\n\n", BuildVersion, BuildDate, runtime.Version())

	if len(os.Args) < 2 {
		printHelp()
		return
	}

	pattern := os.Args[1]
	switch pattern {
	case "dump":
		dumpCache()
		return
	}

	printHelp()
}

func dumpCache() {
	torrents := make(map[string]*cdb.Torrent)
	users := make(map[string]*cdb.User)

	// dump torrent data
	torrentFile, err := os.OpenFile(fmt.Sprintf("%s.gob", cdb.TorrentCacheFile), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Println("Dumping data for torrent cache, this might take a while...")

	err = gob.NewDecoder(torrentFile).Decode(&torrents)
	if err != nil {
		panic(err)
	}

	res, err := json.MarshalIndent(torrents, "", "\t")
	if err != nil {
		panic(err)
	}

	torrentFile, err = os.OpenFile(fmt.Sprintf("%s.gob", cdb.TorrentCacheFile), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	_, err = torrentFile.Write(res)
	if err != nil {
		panic(err)
	}

	err = torrentFile.Close()
	if err != nil {
		panic(err)
	}

	peersLen := 0
	torrentsLen := len(torrents)

	for _, t := range torrents {
		peersLen += len(t.Leechers) + len(t.Seeders)
	}

	fmt.Printf("Done! Exported %d torrent entries with %d peers", torrentsLen, peersLen)

	// dump user data
	userFile, err := os.OpenFile(fmt.Sprintf("%s.gob", cdb.UserCacheFile), os.O_RDONLY, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Println("Dumping data for user cache, this might take a while...")

	err = gob.NewDecoder(userFile).Decode(&users)
	if err != nil {
		panic(err)
	}

	res, err = json.MarshalIndent(users, "", "\t")
	if err != nil {
		panic(err)
	}

	userFile, err = os.OpenFile(fmt.Sprintf("%s.json", cdb.UserCacheFile), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	_, err = userFile.Write(res)
	if err != nil {
		panic(err)
	}

	err = userFile.Close()
	if err != nil {
		panic(err)
	}

	usersLen := len(users)

	fmt.Printf("Done! Exported %d user entries", usersLen)
}

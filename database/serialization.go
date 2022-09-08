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

package database

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	cdb "chihaya/database/types"
	"chihaya/log"
	"chihaya/util"
)

var serializeInterval int

func init() {
	intervals := config.Section("intervals")
	serializeInterval, _ = intervals.GetInt("database_serialize", 68)
}

func (db *Database) startSerializing() {
	go func() {
		for !db.terminate {
			time.Sleep(time.Duration(serializeInterval) * time.Second)
			db.serialize()
		}
	}()
}

func (db *Database) serialize() {
	log.Info.Printf("Serializing database to cache file")

	torrentGobFilename := fmt.Sprintf("%s.gob", cdb.TorrentCacheFile)
	userGobFilename := fmt.Sprintf("%s.gob", cdb.UserCacheFile)

	torrentTmpFilename := fmt.Sprintf("%s.tmp", torrentGobFilename)
	userTmpFilename := fmt.Sprintf("%s.tmp", userGobFilename)

	start := time.Now()

	if func() error {
		torrentFile, err := os.OpenFile(torrentTmpFilename, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Error.Print("Couldn't open file for writing: ", err)
			return err
		}

		//goland:noinspection GoUnhandledErrorResult
		defer func() {
			torrentFile.Sync() //nolint:errcheck
			torrentFile.Close()
		}()

		util.TakeSemaphore(db.TorrentsSemaphore)
		defer util.ReturnSemaphore(db.TorrentsSemaphore)

		if err = gob.NewEncoder(torrentFile).Encode(db.Torrents); err != nil {
			log.Error.Print("Failed to encode torrents for serialization: ", err)
			return err
		}

		return nil
	}() == nil {
		if err := os.Rename(torrentTmpFilename, torrentGobFilename); err != nil {
			log.Error.Print("Couldn't write new torrent cache: ", err)
		}
	}

	if func() error {
		userFile, err := os.OpenFile(userTmpFilename, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Error.Print("Couldn't open file for writing: ", err)
			return err
		}

		//goland:noinspection GoUnhandledErrorResult
		defer func() {
			userFile.Sync() //nolint:errcheck
			userFile.Close()
		}()

		util.TakeSemaphore(db.UsersSemaphore)
		defer util.ReturnSemaphore(db.UsersSemaphore)

		if err = gob.NewEncoder(userFile).Encode(db.Users); err != nil {
			log.Error.Print("Failed to encode users for serialization: ", err)
			return err
		}

		return nil
	}() == nil {
		if err := os.Rename(userTmpFilename, userGobFilename); err != nil {
			log.Error.Print("Couldn't write new user cache: ", err)
		}
	}

	elapsedTime := time.Since(start)
	collectors.UpdateSerializationTime(elapsedTime)
	log.Info.Printf("Done serializing (%s)", elapsedTime.String())
}

func (db *Database) deserialize() {
	log.Info.Print("Deserializing database from cache file...")

	torrentGobFilename := fmt.Sprintf("%s.gob", cdb.TorrentCacheFile)
	userGobFilename := fmt.Sprintf("%s.gob", cdb.UserCacheFile)

	start := time.Now()

	func() {
		torrentFile, err := os.OpenFile(torrentGobFilename, os.O_RDONLY, 0)
		if err != nil {
			log.Warning.Print("Torrent cache missing: ", err)
			return
		}

		//goland:noinspection GoUnhandledErrorResult
		defer torrentFile.Close()

		decoder := gob.NewDecoder(torrentFile)

		util.TakeSemaphore(db.TorrentsSemaphore)
		defer util.ReturnSemaphore(db.TorrentsSemaphore)

		err = decoder.Decode(&db.Torrents)
		if err != nil {
			log.Error.Print("Failed to deserialize torrent cache: ", err)
			return
		}
	}()

	func() {
		userFile, err := os.OpenFile(userGobFilename, os.O_RDONLY, 0)
		if err != nil {
			log.Warning.Print("User cache missing: ", err)
			return
		}

		//goland:noinspection GoUnhandledErrorResult
		defer userFile.Close()

		decoder := gob.NewDecoder(userFile)

		util.TakeSemaphore(db.UsersSemaphore)
		defer util.ReturnSemaphore(db.UsersSemaphore)

		err = decoder.Decode(&db.Users)
		if err != nil {
			log.Error.Print("Failed to deserialize user cache: ", err)
			return
		}
	}()

	util.TakeSemaphore(db.TorrentsSemaphore)
	defer util.ReturnSemaphore(db.TorrentsSemaphore)

	util.TakeSemaphore(db.UsersSemaphore)
	defer util.ReturnSemaphore(db.UsersSemaphore)

	torrents := len(db.Torrents)
	users := len(db.Users)

	peers := 0
	for _, t := range db.Torrents {
		peers += len(t.Leechers) + len(t.Seeders)
	}

	log.Info.Printf("Loaded %d users, %d torrents and %d peers (%s)",
		users, torrents, peers, time.Since(start).String())
}

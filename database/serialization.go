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
	"fmt"
	"os"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	cdb "chihaya/database/types"
	"chihaya/log"
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

	torrentBinFilename := fmt.Sprintf("%s.bin", cdb.TorrentCacheFile)
	userBinFilename := fmt.Sprintf("%s.bin", cdb.UserCacheFile)

	torrentTmpFilename := fmt.Sprintf("%s.tmp", torrentBinFilename)
	userTmpFilename := fmt.Sprintf("%s.tmp", userBinFilename)

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

		if err = cdb.WriteTorrents(torrentFile, *db.Torrents.Load()); err != nil {
			log.Error.Print("Failed to encode torrents for serialization: ", err)
			return err
		}

		return nil
	}() == nil {
		if err := os.Rename(torrentTmpFilename, torrentBinFilename); err != nil {
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

		if err = cdb.WriteUsers(userFile, *db.Users.Load()); err != nil {
			log.Error.Print("Failed to encode users for serialization: ", err)
			return err
		}

		return nil
	}() == nil {
		if err := os.Rename(userTmpFilename, userBinFilename); err != nil {
			log.Error.Print("Couldn't write new user cache: ", err)
		}
	}

	elapsedTime := time.Since(start)
	collectors.UpdateSerializationTime(elapsedTime)
	log.Info.Printf("Done serializing (%s)", elapsedTime.String())
}

func (db *Database) deserialize() {
	log.Info.Print("Deserializing database from cache file...")

	torrentBinFilename := fmt.Sprintf("%s.bin", cdb.TorrentCacheFile)
	userBinFilename := fmt.Sprintf("%s.bin", cdb.UserCacheFile)

	start := time.Now()

	torrents := 0
	peers := 0

	func() {
		torrentFile, err := os.OpenFile(torrentBinFilename, os.O_RDONLY, 0)
		if err != nil {
			log.Warning.Print("Torrent cache missing: ", err)
			return
		}

		//goland:noinspection GoUnhandledErrorResult
		defer torrentFile.Close()

		dbTorrents := make(map[cdb.TorrentHash]*cdb.Torrent)
		if err = cdb.LoadTorrents(torrentFile, dbTorrents); err != nil {
			log.Error.Print("Failed to deserialize torrent cache: ", err)
			return
		}

		torrents = len(dbTorrents)

		for _, t := range dbTorrents {
			peers += int(t.LeechersLength.Load()) + int(t.SeedersLength.Load())
		}

		db.Torrents.Store(&dbTorrents)
	}()

	func() {
		userFile, err := os.OpenFile(userBinFilename, os.O_RDONLY, 0)
		if err != nil {
			log.Warning.Print("User cache missing: ", err)
			return
		}

		//goland:noinspection GoUnhandledErrorResult
		defer userFile.Close()

		users := make(map[string]*cdb.User)
		if err = cdb.LoadUsers(userFile, users); err != nil {
			log.Error.Print("Failed to deserialize user cache: ", err)
			return
		}

		db.Users.Store(&users)
	}()

	users := len(*db.Users.Load())

	log.Info.Printf("Loaded %d users, %d torrents and %d peers (%s)",
		users, torrents, peers, time.Since(start).String())
}

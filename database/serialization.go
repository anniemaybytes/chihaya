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
	"log/slog"
	"os"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	cdb "chihaya/database/types"
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
	slog.Info("serializing database to cache file")

	torrentBinFilename := fmt.Sprintf("%s.bin", cdb.TorrentCacheFile)
	userBinFilename := fmt.Sprintf("%s.bin", cdb.UserCacheFile)

	torrentTmpFilename := fmt.Sprintf("%s.tmp", torrentBinFilename)
	userTmpFilename := fmt.Sprintf("%s.tmp", userBinFilename)

	start := time.Now()

	if func() error {
		torrentFile, err := os.OpenFile(torrentTmpFilename, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			slog.Error("couldn't open file for writing", "err", err, "cdb", cdb.TorrentCacheFile)
			return err
		}

		//goland:noinspection GoUnhandledErrorResult
		defer func() {
			torrentFile.Sync() //nolint:errcheck
			torrentFile.Close()
		}()

		if err = cdb.WriteTorrents(torrentFile, *db.Torrents.Load()); err != nil {
			slog.Error("failed to encode cdb for serialization", "err", err, "cdb", cdb.TorrentCacheFile)
			return err
		}

		return nil
	}() == nil {
		if err := os.Rename(torrentTmpFilename, torrentBinFilename); err != nil {
			slog.Error("couldn't write new cache file", "err", err, "cdb", cdb.TorrentCacheFile)
		}
	}

	if func() error {
		userFile, err := os.OpenFile(userTmpFilename, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			slog.Error("couldn't open file for writing", "err", err, "cdb", cdb.UserCacheFile)
			return err
		}

		//goland:noinspection GoUnhandledErrorResult
		defer func() {
			userFile.Sync() //nolint:errcheck
			userFile.Close()
		}()

		if err = cdb.WriteUsers(userFile, *db.Users.Load()); err != nil {
			slog.Error("failed to encode cdb for serialization", "err", err, "cdb", cdb.UserCacheFile)
			return err
		}

		return nil
	}() == nil {
		if err := os.Rename(userTmpFilename, userBinFilename); err != nil {
			slog.Error("couldn't write new cache file", "err", err, "cdb", cdb.UserCacheFile)
		}
	}

	elapsedTime := time.Since(start)
	collectors.UpdateSerializationTime(elapsedTime)
	slog.Info("done serializing", "elapsed", elapsedTime)
}

func (db *Database) deserialize() {
	slog.Info("deserializing database from cache file")

	torrentBinFilename := fmt.Sprintf("%s.bin", cdb.TorrentCacheFile)
	userBinFilename := fmt.Sprintf("%s.bin", cdb.UserCacheFile)

	var (
		start    = time.Now()
		torrents = 0
		peers    = 0
		users    = 0
	)

	func() {
		torrentFile, err := os.OpenFile(torrentBinFilename, os.O_RDONLY, 0)
		if err != nil {
			slog.Warn("cache file missing", "err", err, "cdb", cdb.TorrentCacheFile)
			return
		}

		//goland:noinspection GoUnhandledErrorResult
		defer torrentFile.Close()

		dbTorrents := make(map[cdb.TorrentHash]*cdb.Torrent)
		if err = cdb.LoadTorrents(torrentFile, dbTorrents); err != nil {
			slog.Warn("failed to deserialize cache", "err", err, "cdb", cdb.TorrentCacheFile)
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
			slog.Warn("cache file missing", "err", err, "cdb", cdb.UserCacheFile)
			return
		}

		//goland:noinspection GoUnhandledErrorResult
		defer userFile.Close()

		dbUsers := make(map[string]*cdb.User)
		if err = cdb.LoadUsers(userFile, dbUsers); err != nil {
			slog.Warn("failed to deserialize cache", "err", err, "cdb", cdb.UserCacheFile)
			return
		}

		users = len(dbUsers)

		db.Users.Store(&dbUsers)
	}()

	slog.Info("deserialization complete", "elapsed", time.Since(start),
		"users", users, "torrents", torrents, "peers", peers)
}

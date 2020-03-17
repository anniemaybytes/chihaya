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

package server

import (
	"bytes"
	"chihaya/config"
	"chihaya/database"
	cdb "chihaya/database/types"
	"time"

	"github.com/zeebo/bencode"
)

func writeScrapeInfo(torrent *cdb.Torrent) map[string]interface{} {
	ret := make(map[string]interface{})
	ret["complete"] = len(torrent.Seeders)
	ret["downloaded"] = torrent.Snatched
	ret["incomplete"] = len(torrent.Leechers)

	return ret
}

func scrape(params *queryParams, db *database.Database, buf *bytes.Buffer) {
	if !config.GetBool("scrape", true) {
		failure("Scrape convention is not supported", buf, 1*time.Hour)
		return
	}

	scrapeData := make(map[string]interface{})
	fileData := make(map[string]interface{})

	if params.infoHashes != nil {
		db.TorrentsMutex.RLock()

		for _, infoHash := range params.infoHashes {
			torrent, exists := db.Torrents[infoHash]
			if exists {
				fileData[infoHash] = writeScrapeInfo(torrent)
			}
		}

		db.TorrentsMutex.RUnlock()
	} else if infoHash, exists := params.get("info_hash"); exists {
		db.TorrentsMutex.RLock()

		torrent, exists := db.Torrents[infoHash]
		if exists {
			fileData[infoHash] = writeScrapeInfo(torrent)
		}

		db.TorrentsMutex.RUnlock()
	} else {
		scrapeData["failure reason"] = "Scrape without info_hash is not supported"
	}

	scrapeData["files"] = fileData
	scrapeData["flags"] = map[string]interface{}{
		"min_request_interval": config.MinScrapeInterval / time.Second, // Assuming in seconds
	}
	// the following are for compatibility with clients that don't implement scrape flags
	scrapeData["interval"] = config.MinScrapeInterval / time.Second     // Assuming in seconds
	scrapeData["min interval"] = config.MinScrapeInterval / time.Second // Assuming in seconds

	bufdata, err := bencode.EncodeBytes(scrapeData)
	if err != nil {
		panic(err)
	}

	buf.Write(bufdata)
}

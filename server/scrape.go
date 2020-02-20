// +build scrape

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
	cdb "chihaya/database"
	"github.com/zeebo/bencode"
)

func writeScrapeInfo(torrent *cdb.Torrent) map[string]interface{} {
	ret := make(map[string]interface{})
	ret["complete"] = len(torrent.Seeders)
	ret["downloaded"] = torrent.Snatched
	ret["incomplete"] = len(torrent.Leechers)

	return ret
}

func scrape(params *queryParams, db *cdb.Database, buf *bytes.Buffer) {
	scrapeData := make(map[string]interface{})
	fileData := make(map[string]interface{})

	db.TorrentsMutex.RLock()

	if params.infoHashes != nil {
		for _, infoHash := range params.infoHashes {
			torrent, exists := db.Torrents[infoHash]
			if exists {
				fileData[infoHash] = writeScrapeInfo(torrent)
			}
		}
	} else if infoHash, exists := params.get("info_hash"); exists {
		torrent, exists := db.Torrents[infoHash]
		if exists {
			fileData[infoHash] = writeScrapeInfo(torrent)
		}
	}
	db.TorrentsMutex.RUnlock()

	scrapeData["files"] = fileData

	bufdata, err := bencode.EncodeBytes(scrapeData)
	if err != nil {
		panic(err)
	}

	buf.Write(bufdata)
}

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
	"chihaya/server/params"

	"github.com/valyala/fasthttp"
	"github.com/zeebo/bencode"
)

var scrapeInterval int

func init() {
	intervals := config.Section("intervals")
	scrapeInterval, _ = intervals.GetInt("scrape", 900)
}

func scrape(ctx *fasthttp.RequestCtx, user *cdb.User, db *database.Database, buf *bytes.Buffer) int {
	qp, err := params.ParseQuery(string(ctx.Request.URI().QueryString()))
	if err != nil {
		panic(err)
	}

	scrapeData := make(map[string]interface{})
	fileData := make(map[cdb.TorrentHash]interface{})

	dbTorrents := *db.Torrents.Load()

	if qp.InfoHashes() != nil {
		for _, infoHash := range qp.InfoHashes() {
			torrent, exists := dbTorrents[infoHash]
			if exists {
				if !isDisabledDownload(db, user, torrent) {
					ret := make(map[string]interface{})
					ret["complete"] = torrent.SeedersLength.Load()
					ret["downloaded"] = torrent.Snatched.Load()
					ret["incomplete"] = torrent.LeechersLength.Load()

					fileData[infoHash] = ret
				}
			}
		}
	} else {
		scrapeData["failure reason"] = "Scrape without info_hash is not supported"
	}

	scrapeData["files"] = fileData
	scrapeData["flags"] = map[string]interface{}{
		"min_request_interval": scrapeInterval,
	}
	// the following are for compatibility with clients that don't implement scrape flags
	scrapeData["interval"] = scrapeInterval
	scrapeData["min interval"] = scrapeInterval

	encoder := bencode.NewEncoder(buf)
	if err = encoder.Encode(scrapeData); err != nil {
		panic(err)
	}

	return fasthttp.StatusOK
}

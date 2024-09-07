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
	qp, err := params.ParseQuery(ctx.Request.URI().QueryArgs())
	if err != nil {
		panic(err)
	}

	response := make(map[string]interface{})
	filesList := make(map[cdb.TorrentHash]interface{})

	dbTorrents := *db.Torrents.Load()

	if len(qp.Params.InfoHashes) > 0 {
		for _, infoHash := range qp.Params.InfoHashes {
			torrent, exists := dbTorrents[infoHash]
			if exists {
				if !isDisabledDownload(db, user, torrent) {
					fileMap := make(map[string]interface{})
					fileMap["complete"] = torrent.SeedersLength.Load()
					fileMap["downloaded"] = torrent.Snatched.Load()
					fileMap["incomplete"] = torrent.LeechersLength.Load()

					filesList[infoHash] = fileMap
				}
			}
		}
	} else {
		failure("Unsupported request - must provide at least one info_hash", buf, 0)
		return fasthttp.StatusOK // Required by torrent clients to interpret failure response
	}

	response["files"] = filesList
	response["flags"] = map[string]interface{}{
		"min_request_interval": scrapeInterval,
	}

	encoder := bencode.NewEncoder(buf)
	if err = encoder.Encode(response); err != nil {
		panic(err)
	}

	return fasthttp.StatusOK
}

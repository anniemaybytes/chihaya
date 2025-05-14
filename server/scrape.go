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
	"chihaya/util"

	"github.com/valyala/fasthttp"
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

	if len(qp.Params.InfoHashes) > 0 {
		util.BencodeScrapeHeader(buf)

		// pre-sort keys
		util.BencodeSortTorrentHashKeys(qp.Params.InfoHashes)

		dbTorrents := *db.Torrents.Load()

		for _, infoHash := range qp.Params.InfoHashes {
			if torrent, exists := dbTorrents[infoHash]; exists {
				if !isDisabledDownload(db, user, torrent) {
					util.BencodeScrapeTorrent(buf, infoHash,
						int64(torrent.SeedersLength.Load()),
						int64(torrent.Snatched.Load()),
						int64(torrent.LeechersLength.Load()),
					)
				}
			}
		}

		util.BencodeScrapeFooter(buf, scrapeInterval)

		return fasthttp.StatusOK
	}

	failure("Unsupported request - must provide at least one info_hash", buf, 0)

	return fasthttp.StatusOK // Required by torrent clients to interpret failure response
}

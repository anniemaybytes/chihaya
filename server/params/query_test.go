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

package params

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"testing"

	cdb "chihaya/database/types"
	"chihaya/util"

	"github.com/valyala/fasthttp"
)

var infoHashes []cdb.TorrentHash

func TestMain(m *testing.M) {
	var token cdb.TorrentHash

	for i := 0; i < 10; i++ {
		_, _ = util.UnsafeReadRand(token[:])

		infoHashes = append(infoHashes, token)
	}

	os.Exit(m.Run())
}

func TestParseQuery(t *testing.T) {
	var queryParsed QueryParam
	queryParsed.Params.Event, queryParsed.Exists.Event = "completed", true
	queryParsed.Params.Port, queryParsed.Exists.Port = 25362, true
	queryParsed.Params.PeerID, queryParsed.Exists.PeerID = "-CH010-VnpZR7uz31I1A", true
	queryParsed.Params.Left, queryParsed.Exists.Left = 0, true

	query := fmt.Sprintf("event=%s&port=%d&peer_id=%s&left=%d",
		queryParsed.Params.Event,
		queryParsed.Params.Port,
		queryParsed.Params.PeerID,
		queryParsed.Params.Left,
	)

	for _, infoHash := range infoHashes {
		queryParsed.Params.InfoHashes = append(queryParsed.Params.InfoHashes, infoHash)
		queryParsed.Exists.InfoHashes = true
		query += "&info_hash=" + url.QueryEscape(string(infoHash[:]))
	}

	args := fasthttp.Args{}
	args.Parse(query)

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(qp, queryParsed) {
		t.Fatalf("Parsed query map (%v) is not deeply equal as original (%v)!", qp, queryParsed)
	}
}

func TestBrokenParseQuery(t *testing.T) {
	var brokenQueryParsed QueryParam
	brokenQueryParsed.Params.Event, brokenQueryParsed.Exists.Event = "started", true
	brokenQueryParsed.Params.IP, brokenQueryParsed.Exists.IP = "", true

	args := fasthttp.Args{}
	args.Parse("event=started&ip=")

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(qp, brokenQueryParsed) {
		t.Fatalf("Parsed query map (%v) is not deeply equal as original (%v)!", qp, brokenQueryParsed)
	}
}

func TestLowerKey(t *testing.T) {
	args := fasthttp.Args{}
	args.Parse("EvEnT=c0mPl3tED")

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if param, exists := qp.Params.Event, qp.Exists.Event; !exists || param != "c0mPl3tED" {
		t.Fatalf("Got parsed value %s but expected c0mPl3tED for \"event\"!", param)
	}
}

func TestUnescape(t *testing.T) {
	args := fasthttp.Args{}
	args.Parse("%21%40%23=%24%25%5E")

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if param, exists := qp.Params.testGarbageUnescape, qp.Exists.testGarbageUnescape; !exists || param != "$%^" {
		t.Fatal(fmt.Sprintf("Got parsed value %s but expected", param), "$%^ for \"!@#\"!")
	}
}

func TestString(t *testing.T) {
	args := fasthttp.Args{}
	args.Parse("event=completed")

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if param, exists := qp.Params.Event, qp.Exists.Event; !exists || param != "completed" {
		t.Fatalf("Got parsed value %s but expected completed for \"event\"!", param)
	}
}

func TestGetUint64(t *testing.T) {
	val := uint64(1<<62 + 42)

	args := fasthttp.Args{}
	args.Parse("left=" + strconv.FormatUint(val, 10))

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if parsedVal, exists := qp.Params.Left, qp.Exists.Left; !exists || parsedVal != val {
		t.Fatalf("Got parsed value %v but expected %v for \"left\"!", parsedVal, val)
	}
}

func TestGetUint16(t *testing.T) {
	val := uint16(1<<15 + 4242)

	args := fasthttp.Args{}
	args.Parse("port=" + strconv.FormatUint(uint64(val), 10))

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if parsedVal, exists := qp.Params.Port, qp.Exists.Port; !exists || parsedVal != val {
		t.Fatalf("Got parsed value %v but expected %v for \"port\"!", parsedVal, val)
	}
}

func TestInfoHashes(t *testing.T) {
	query := ""

	for _, infoHash := range infoHashes {
		query += "info_hash=" + url.QueryEscape(string(infoHash[:])) + "&"
	}

	query = query[:len(query)-1]

	args := fasthttp.Args{}
	args.Parse(query)

	qp, err := ParseQuery(&args)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(qp.Params.InfoHashes, infoHashes) {
		t.Fatalf("Parsed info hashes (%v) are not deeply equal as original (%v)!", qp.Params.InfoHashes, infoHashes)
	}
}

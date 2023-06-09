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

	"chihaya/util"
)

var infoHashes []string

func TestMain(m *testing.M) {
	var token []byte

	for i := 0; i < 10; i++ {
		token = make([]byte, 20)
		_, _ = util.UnsafeReadRand(token[:])

		infoHashes = append(infoHashes, string(token))
	}

	os.Exit(m.Run())
}

func TestParseQuery(t *testing.T) {
	query := ""

	for _, infoHash := range infoHashes {
		query += "info_hash=" + url.QueryEscape(infoHash) + "&"
	}

	queryMap := make(map[string]string)
	queryMap["event"] = "completed"
	queryMap["port"] = "25362"
	queryMap["peer_id"] = "-CH010-VnpZR7uz31I1A"
	queryMap["left"] = "0"

	for k, v := range queryMap {
		query += k + "=" + v + "&"
	}

	query = query[:len(query)-1]

	qp, err := ParseQuery(query)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(qp.params, queryMap) {
		t.Fatalf("Parsed query map (%v) is not deeply equal as original (%v)!", qp.params, queryMap)
	}

	if !reflect.DeepEqual(qp.infoHashes, infoHashes) {
		t.Fatalf("Parsed info hashes (%v) are not deeply equal as original (%v)!", qp.infoHashes, infoHashes)
	}
}

func TestBrokenParseQuery(t *testing.T) {
	brokenQueryMap := make(map[string]string)
	brokenQueryMap["event"] = "started"
	brokenQueryMap["bug"] = ""
	brokenQueryMap["yes"] = ""

	qp, err := ParseQuery("event=started&bug=&yes=")
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(qp.params, brokenQueryMap) {
		t.Fatalf("Parsed query map (%v) is not deeply equal as original (%v)!", qp.params, brokenQueryMap)
	}
}

func TestLowerKey(t *testing.T) {
	qp, err := ParseQuery("EvEnT=c0mPl3tED")
	if err != nil {
		panic(err)
	}

	if param, exists := qp.Get("event"); !exists || param != "c0mPl3tED" {
		t.Fatalf("Got parsed value %s but expected c0mPl3tED for \"event\"!", param)
	}
}

func TestUnescape(t *testing.T) {
	qp, err := ParseQuery("%21%40%23=%24%25%5E")
	if err != nil {
		panic(err)
	}

	if param, exists := qp.Get("!@#"); !exists || param != "$%^" {
		t.Fatal(fmt.Sprintf("Got parsed value %s but expected", param), "$%^ for \"!@#\"!")
	}
}

func TestGet(t *testing.T) {
	qp, err := ParseQuery("event=completed")
	if err != nil {
		panic(err)
	}

	if param, exists := qp.Get("event"); !exists || param != "completed" {
		t.Fatalf("Got parsed value %s but expected completed for \"event\"!", param)
	}
}

func TestGetUint64(t *testing.T) {
	val := uint64(1<<62 + 42)

	qp, err := ParseQuery("left=" + strconv.FormatUint(val, 10))
	if err != nil {
		panic(err)
	}

	if parsedVal, exists := qp.GetUint64("left"); !exists || parsedVal != val {
		t.Fatalf("Got parsed value %v but expected %v for \"left\"!", parsedVal, val)
	}
}

func TestGetUint16(t *testing.T) {
	val := uint16(1<<15 + 4242)

	qp, err := ParseQuery("port=" + strconv.FormatUint(uint64(val), 10))
	if err != nil {
		panic(err)
	}

	if parsedVal, exists := qp.GetUint16("port"); !exists || parsedVal != val {
		t.Fatalf("Got parsed value %v but expected %v for \"port\"!", parsedVal, val)
	}
}

func TestInfoHashes(t *testing.T) {
	query := ""

	for _, infoHash := range infoHashes {
		query += "info_hash=" + url.QueryEscape(infoHash) + "&"
	}

	query = query[:len(query)-1]

	qp, err := ParseQuery(query)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(qp.InfoHashes(), infoHashes) {
		t.Fatalf("Parsed info hashes (%v) are not deeply equal as original (%v)!", qp.InfoHashes(), infoHashes)
	}
}

func TestRawQuery(t *testing.T) {
	q := "event=completed&port=25541&left=0&uploaded=0&downloaded=0"

	qp, err := ParseQuery(q)
	if err != nil {
		panic(err)
	}

	if rq := qp.RawQuery(); rq != q {
		t.Fatalf("Got raw query %s but expected %s", rq, q)
	}
}

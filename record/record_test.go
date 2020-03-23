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

package record

import (
	"bufio"
	"chihaya/util"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type record struct {
	port      uint16
	uid       uint32
	tid       uint32
	absup     uint64
	up, down  int64
	event, ip string
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	tempPath := filepath.Join(os.TempDir(), "chihaya_record-"+util.RandStringBytes(6))

	err := os.Mkdir(tempPath, 0755)
	if err != nil {
		panic(err)
	}

	err = os.Chdir(tempPath)
	if err != nil {
		panic(err)
	}

	enabledByDefault = true // force-enable for tests

	Init()

	os.Exit(m.Run())
}

func TestRecord(t *testing.T) {
	var (
		recordValues    []record
		expectedOutputs []string
	)

	for i := 0; i < 10; i++ {
		tmp := record{
			uint16(rand.Uint32()),
			rand.Uint32(),
			rand.Uint32(),
			rand.Uint64(),
			int64(rand.Uint64()),
			int64(rand.Uint64()),
			"completed",
			"127.0.0.1",
		}
		recordValues = append(recordValues, tmp)
		expectedOutputs = append(
			expectedOutputs,
			"["+
				strconv.FormatUint(uint64(tmp.tid), 10)+","+
				strconv.FormatUint(uint64(tmp.uid), 10)+","+
				strconv.FormatInt(tmp.up, 10)+","+
				strconv.FormatInt(tmp.down, 10)+","+
				strconv.FormatUint(tmp.absup, 10)+","+
				"\""+tmp.event+"\""+","+
				"\""+tmp.ip+"\","+
				strconv.FormatUint(uint64(tmp.port), 10)+
				"]",
		)
	}

	for _, item := range recordValues {
		Record(item.tid, item.uid, item.up, item.down, item.absup, item.event, item.ip, item.port)
	}

	time.Sleep(200 * time.Millisecond)

	// In theory, below line can fail if this line was called in a different hour than when the file was made
	// In practice, this would never occur since the file should be made fast enough for it to be in same error.
	recordFile, err := openEventFile(time.Now())
	if err != nil {
		t.Fatalf("Faced error in opening file: %s", err)
	}

	recordScanner := bufio.NewScanner(recordFile)
	recordScanner.Split(bufio.ScanLines)

	var recordLines []string

	for recordScanner.Scan() {
		recordLines = append(recordLines, recordScanner.Text())
	}

	if err := recordScanner.Err(); err != nil {
		t.Fatalf("Faced error in reading: %s", err)
	}

	if len(expectedOutputs) != len(recordLines) {
		t.Fatalf("The number of records do not match with what is expected! (expected %d, got %d)",
			len(expectedOutputs), len(recordLines))
	}

	for index, recordLine := range recordLines { // noinspection GoNilness
		if expectedOutputs[index] != recordLine {
			t.Fatalf("Expected %s but got %s in record!", expectedOutputs[index], recordLine)
		}
	}
}

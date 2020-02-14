// +build record

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
	"bufio"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

type Record struct {
	tid, uid  uint64
	up, down  int64
	absup     uint64
	event, ip string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestRecord(t *testing.T) {
	var recordValues []Record
	var expectedOutputs []string
	for i := 0; i < 10; i++ {
		tmp := Record{rand.Uint64(), rand.Uint64(),
			int64(rand.Uint64()), int64(rand.Uint64()),
			rand.Uint64(), "completed", "127.0.0.1"}
		recordValues = append(recordValues, tmp)
		expectedOutputs = append(
			expectedOutputs,
			"["+
				strconv.FormatUint(tmp.tid, 10)+","+
				strconv.FormatUint(tmp.uid, 10)+","+
				strconv.FormatInt(tmp.up, 10)+","+
				strconv.FormatInt(tmp.down, 10)+","+
				strconv.FormatUint(tmp.absup, 10)+","+
				"\""+tmp.event+"\""+","+
				"\""+tmp.ip+"\""+
				"]",
		)
	}
	for _, item := range recordValues {
		record(item.tid, item.uid, item.up, item.down, item.absup, item.event, item.ip)
	}
	// In theory, below line can fail if this line was called in a different hour than when the file was made
	// In practice, this would never occur since the file should be made fast enough for it to be in same error.
	recordFile, err := os.Open("events/events_" + time.Now().Format("2006-01-02T15") + ".json")
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
		t.Fatalf("The number of records do not match with what is expected!")
	}
	for index, recordLine := range recordLines {
		if expectedOutputs[index] != recordLine {
			t.Fatalf("Expected %s but got %s in record!", expectedOutputs[index], recordLine)
		}
	}
	errRemove := os.RemoveAll("events") // Cleanup
	if errRemove != nil && !os.IsNotExist(errRemove) {
		t.Fatalf("Cannot remove existing events directory!")
	}
}

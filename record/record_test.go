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
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	cdb "chihaya/database/types"
	"chihaya/util"
)

type record struct {
	port           uint16
	uid, tid       uint32
	up, down, left uint64
	event, ip      string
}

func TestMain(m *testing.M) {
	path, err := os.MkdirTemp(os.TempDir(), "chihaya_record-*")
	if err != nil {
		panic(err)
	}

	if err = os.Chmod(path, 0755); err != nil {
		panic(err)
	}

	if err = os.Chdir(path); err != nil {
		panic(err)
	}

	enabled = true // force-enable for tests

	os.Exit(m.Run())
}

func TestRecord(t *testing.T) {
	var (
		values  []record
		outputs = []string{header}
	)

	for i := 0; i < 10; i++ {
		entry := record{
			uint16(util.UnsafeUint32()),
			util.UnsafeUint32(),
			util.UnsafeUint32(),
			util.UnsafeUint64(),
			util.UnsafeUint64(),
			util.UnsafeUint64(),
			"completed",
			"127.0.0.1",
		}
		values = append(values, entry)
		outputs = append(
			outputs,
			strings.Join([]string{
				strconv.FormatUint(uint64(entry.tid), 10),
				strconv.FormatUint(uint64(entry.uid), 10),
				entry.ip + ":" + strconv.FormatUint(uint64(entry.port), 10),
				entry.event,
				strconv.FormatUint(entry.up, 10),
				strconv.FormatUint(entry.down, 10),
				strconv.FormatUint(entry.left, 10),
			}, ","),
		)
	}

	for _, entry := range values {
		Record(
			entry.tid,
			entry.uid,
			cdb.NewPeerAddressFromIPPort(net.ParseIP(entry.ip).To4(), entry.port),
			entry.event,
			entry.up,
			entry.down,
			entry.left)
	}

	time.Sleep(200 * time.Millisecond)

	// In theory, below line can fail if this line was called in a different hour than when the file was made
	// In practice, this would never occur since the file should be made fast enough for it to be in same error.
	file, err := os.OpenFile(filename(time.Now()), os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Faced error in opening file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var recordLines []string

	for scanner.Scan() {
		recordLines = append(recordLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Faced error in reading: %s", err)
	}

	if len(outputs) != len(recordLines) {
		t.Fatalf("The number of records do not match with what is expected! (expected %d, got %d)",
			len(outputs), len(recordLines))
	}

	for index, recordLine := range recordLines { // noinspection GoNilness
		if outputs[index] != recordLine {
			t.Fatalf("Expected %s but got %s in record!", outputs[index], recordLine)
		}
	}
}

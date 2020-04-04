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
	"bytes"
	"chihaya/config"
	"chihaya/util"
	"os"
	"strconv"
	"time"
)

var enabledByDefault = false

var recordChan chan []byte

func openEventFile(t time.Time) (*os.File, error) {
	return os.OpenFile("events/events_"+t.Format("2006-01-02T15")+".json", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
}

func Init() {
	enabledByDefault, _ := config.GetBool("record", enabledByDefault)
	if !enabledByDefault {
		return
	}

	err := os.Mkdir("events", 0755)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	start := time.Now()
	recordChan = make(chan []byte)

	recordFile, err := openEventFile(start)
	if err != nil {
		panic(err)
	}

	go func() {
		for buf := range recordChan {
			now := time.Now()
			if now.Hour() != start.Hour() {
				start = now

				err := recordFile.Close()
				if err != nil {
					panic(err)
				}

				recordFile, err = openEventFile(start)
				if err != nil {
					panic(err)
				}
			}

			_, err := recordFile.Write(buf)
			if err != nil {
				panic(err)
			}
		}
	}()
}

func Record(
	tid,
	uid uint32,
	ip string,
	port uint16,
	event string,
	seeding bool,
	rawUp,
	rawDown int64,
	up,
	down,
	left uint64) {
	enabledByDefault, _ := config.GetBool("record", enabledByDefault)
	if !enabledByDefault {
		return
	}

	if up == 0 && down == 0 {
		return
	}

	b := make([]byte, 0, 64)
	buf := bytes.NewBuffer(b)

	buf.WriteString("[")
	buf.WriteString(strconv.FormatUint(uint64(tid), 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(uint64(uid), 10))
	buf.WriteString(",\"")
	buf.WriteString(ip)
	buf.WriteString("\",")
	buf.WriteString(strconv.FormatUint(uint64(port), 10))
	buf.WriteString(",\"")
	buf.WriteString(event)
	buf.WriteString("\",")
	buf.WriteString(util.Btoa(seeding))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatInt(rawUp, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatInt(rawDown, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(up, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(down, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(left, 10))
	buf.WriteString("]\n")

	recordChan <- buf.Bytes()
}

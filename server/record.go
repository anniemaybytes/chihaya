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
	"bytes"
	"os"
	"strconv"
	"time"
)

var recordChan chan []byte

func openEventFile(t time.Time) (*os.File, error) {
	return os.OpenFile("events/events_"+t.Format("2006-01-02T15")+".json", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
}

func init() {
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
				recordFile.Close()
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

func record(tid, uid uint64, up, down int64, absup uint64, event, ip string) {
	if up == 0 && down == 0 {
		return
	}
	b := make([]byte, 0, 64)
	buf := bytes.NewBuffer(b)

	buf.WriteString("[")
	buf.WriteString(strconv.FormatUint(tid, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(uid, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatInt(up, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatInt(down, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(absup, 10))
	buf.WriteString(",\"")
	buf.WriteString(event)
	buf.WriteString("\",\"")
	buf.WriteString(ip)
	buf.WriteString("\"]\n")

	recordChan <- buf.Bytes()
}

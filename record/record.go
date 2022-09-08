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
	"os"
	"strconv"
	"time"

	"chihaya/config"
	"chihaya/log"
	"chihaya/util"
)

var enabled = false
var initialized = false
var recordChan chan []byte

func getFile(t time.Time) (*os.File, error) {
	return os.OpenFile("events/events_"+t.Format("2006-01-02T15")+".json", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
}

func Init() {
	if enabled, _ := config.GetBool("record", enabled); !enabled {
		return
	}

	if err := os.Mkdir("events", 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}

	start := time.Now()
	recordChan = make(chan []byte)

	recordFile, err := getFile(start)
	if err != nil {
		panic(err)
	}

	go func() {
		for buf := range recordChan {
			now := time.Now()
			if now.Hour() != start.Hour() {
				start = now

				if err := recordFile.Close(); err != nil {
					panic(err)
				}

				recordFile, err = getFile(start)
				if err != nil {
					panic(err)
				}
			}

			if _, err := recordFile.Write(buf); err != nil {
				panic(err)
			}
		}
	}()

	initialized = true
}

func Record(
	tid,
	uid uint32,
	ip string,
	port uint16,
	event string,
	seeding bool,
	deltaUp,
	deltaDown int64,
	up,
	down,
	left uint64) {
	if enabled, _ := config.GetBool("record", enabled); !enabled {
		return
	}

	if !initialized {
		log.Fatal.Fatalln("Can not Record without prior initialization")
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
	buf.WriteString(strconv.FormatInt(deltaUp, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatInt(deltaDown, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(up, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(down, 10))
	buf.WriteString(",")
	buf.WriteString(strconv.FormatUint(left, 10))
	buf.WriteString("]\n")

	recordChan <- buf.Bytes()
}

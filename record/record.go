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
	cdb "chihaya/database/types"
	"chihaya/util"
)

var (
	enabled     = false // global for testing purposes
	initialized = false
	channel     chan []byte
)

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
	channel = make(chan []byte)

	file, err := getFile(start)
	if err != nil {
		panic(err)
	}

	go func() {
		for buf := range channel {
			now := time.Now()
			if now.Hour() != start.Hour() {
				start = now

				if err = file.Close(); err != nil {
					panic(err)
				}

				file, err = getFile(start)
				if err != nil {
					panic(err)
				}
			}

			if _, err = file.Write(buf); err != nil {
				panic(err)
			}
		}
	}()

	initialized = true
}

func Record(tid, uid uint32, addr cdb.PeerAddress, event string, seeding bool, deltaUp, deltaDown int64,
	up, down, left uint64) {
	if enabled, _ := config.GetBool("record", enabled); !enabled {
		return
	}

	if !initialized {
		panic("can not Record without prior initialization")
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
	buf.WriteString(addr.IPString())
	buf.WriteString("\",")
	buf.WriteString(strconv.FormatUint(uint64(addr.Port()), 10))
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

	channel <- buf.Bytes()
}

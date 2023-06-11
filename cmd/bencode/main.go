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

package main

import (
	"encoding/json"
	"fmt"
	"github.com/zeebo/bencode"
	"os"
	"runtime"
)

// provided at compile-time
var (
	BuildDate    = "0000-00-00T00:00:00+0000"
	BuildVersion = "development"
)

func help() {
	fmt.Printf("bencode for chihaya (kuroneko), ver=%s date=%s runtime=%s\n\n",
		BuildVersion, BuildDate, runtime.Version())
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Println("  decode  decode bencoded string into json object")
	fmt.Println("  encode  encode json object into bencoded string")
}

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}

	switch os.Args[1] {
	case "decode":
		var val interface{}

		decoder := bencode.NewDecoder(os.Stdin)
		if err := decoder.Decode(&val); err != nil {
			panic(err)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "\t")

		if err := encoder.Encode(val); err != nil {
			panic(err)
		}
	case "encode":
		var val interface{}

		decoder := json.NewDecoder(os.Stdin)
		decoder.UseNumber()

		if err := decoder.Decode(&val); err != nil {
			panic(err)
		}

		encoder := bencode.NewEncoder(os.Stdout)
		if err := encoder.Encode(val); err != nil {
			panic(err)
		}
	default:
		help()
	}
}

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
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/zeebo/bencode"
)

var (
	decode, help bool
)

// provided at compile-time
var (
	BuildDate    = "0000-00-00T00:00:00+0000"
	BuildVersion = "development"
)

func init() {
	flag.BoolVar(&decode, "d", false, "Decodes data instead of encoding")
	flag.BoolVar(&help, "h", false, "Prints this help message")
}

func main() {
	fmt.Printf("bencode for chihaya (kuroneko), ver=%s date=%s runtime=%s\n\n",
		BuildVersion, BuildDate, runtime.Version())

	flag.Parse()

	if help {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()

		return
	}

	var val interface{}

	if decode {
		decoder := bencode.NewDecoder(os.Stdin)

		err := decoder.Decode(&val)
		if err != nil {
			panic(err)
		}

		out, err := json.MarshalIndent(val, "", "\t")
		if err != nil {
			panic(err)
		}

		fmt.Print(string(out))
	} else {
		decoder := json.NewDecoder(os.Stdin)
		decoder.UseNumber()

		err := decoder.Decode(&val)
		if err != nil {
			panic(err)
		}

		encoder := bencode.NewEncoder(os.Stdout)
		err = encoder.Encode(val)
		if err != nil {
			panic(err)
		}
	}
}

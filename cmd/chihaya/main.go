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
	"chihaya/log"
	"chihaya/server"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
)

var profile, help bool

// provided at compile-time
var (
	BuildDate    = "0000-00-00T00:00:00+0000"
	BuildVersion = "development"
)

func init() {
	flag.BoolVar(&profile, "P", false, "Generate profiling data for pprof into chihaya.cpu")
	flag.BoolVar(&help, "h", false, "Shows this help dialog")
}

func main() {
	fmt.Printf("chihaya (kuroneko), ver=%s date=%s runtime=%s\n\n", BuildVersion, BuildDate, runtime.Version())

	flag.Parse()

	if help {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()

		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	if profile {
		log.Info.Printf("Running with profiling enabled, found %d CPUs", runtime.NumCPU())

		f, err := os.Create("chihaya.cpu")
		if err != nil {
			log.Fatal.Fatalf("Failed to create profile file: %s\n", err)
		} else {
			err = pprof.StartCPUProfile(f)
			if err != nil {
				log.Fatal.Fatalf("Can not start profiling session: %s\n", err)
			}
		}
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c

		if profile {
			pprof.StopCPUProfile()
		}

		log.Info.Println("Caught interrupt, shutting down...")
		server.Stop()
		<-c
		os.Exit(0)
	}()

	server.Start()
}

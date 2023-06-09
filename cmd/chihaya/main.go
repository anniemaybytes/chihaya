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
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"chihaya/log"
	"chihaya/server"
)

var (
	pprof string
	help  bool
)

// Provided at compile-time
var (
	BuildDate    = "0000-00-00T00:00:00+0000"
	BuildVersion = "development"
)

func init() {
	flag.StringVar(&pprof, "P", "", "Starts special pprof debug server on specified addr")
	flag.BoolVar(&help, "h", false, "Shows this help dialog")
}

func main() {
	fmt.Printf("chihaya (kuroneko), ver=%s date=%s runtime=%s, cpus=%d\n\n",
		BuildVersion, BuildDate, runtime.Version(), runtime.GOMAXPROCS(0))

	flag.Parse()

	if help {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()

		return
	}

	if len(pprof) > 0 {
		// Both are disabled by default; sample 1% of events
		runtime.SetMutexProfileFraction(100)
		runtime.SetBlockProfileRate(100)

		go func() {
			l, err := net.Listen("tcp", pprof)
			if err != nil {
				log.Error.Printf("Failed to start special pprof debug server: %v", err)
				return
			}

			//nolint:gosec
			s := &http.Server{
				Handler: http.DefaultServeMux,
			}

			log.Warning.Printf("Started special pprof debug server on %s", l.Addr())

			_ = s.Serve(l)
		}()
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c

		log.Info.Print("Caught interrupt, shutting down...")

		server.Stop()
		<-c
		os.Exit(0)
	}()

	log.Info.Print("Starting main server loop...")
	server.Start()
}

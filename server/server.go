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
	"chihaya/collectors"
	"chihaya/config"
	"chihaya/database"
	"chihaya/log"
	"chihaya/record"
	"chihaya/util"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/zeebo/bencode"
)

type httpHandler struct {
	terminate bool

	waitGroup sync.WaitGroup

	// Internal stats
	requests uint64

	bufferPool       *util.BufferPool
	db               *database.Database
	normalRegisterer prometheus.Registerer
	normalCollector  *collectors.NormalCollector
	adminCollector   *collectors.AdminCollector

	startTime time.Time
}

var (
	handler  *httpHandler
	listener net.Listener
)

func failure(err string, buf io.Writer, interval time.Duration) {
	failureData := make(map[string]interface{})
	failureData["failure reason"] = err
	failureData["interval"] = interval / time.Second     // Assuming in seconds
	failureData["min interval"] = interval / time.Second // Assuming in seconds

	data, errz := bencode.EncodeBytes(failureData)
	if errz != nil {
		panic(errz)
	}

	if _, errz = buf.Write(data); errz != nil {
		panic(errz)
	}
}

func (handler *httpHandler) respond(r *http.Request, buf io.Writer) bool {
	dir, action := path.Split(r.URL.Path)
	if action == "" {
		return false
	}

	// Handle public endpoints (/:action)

	passkey := path.Dir(dir)[1:]
	if passkey == "" {
		switch action {
		case "check":
			_, _ = io.WriteString(buf, fmt.Sprintf("%d", time.Now().Unix()))
			return true
		}

		return false
	}

	// Handle private endpoints (/:passkey/:action)

	handler.db.UsersMutex.RLock()
	user, exists := handler.db.Users[passkey]
	handler.db.UsersMutex.RUnlock()

	if !exists {
		failure("Your passkey is invalid", buf, 1*time.Hour)
		return true
	}

	switch action {
	case "announce":
		announce(r.URL.RawQuery, r.Header, r.RemoteAddr, user, handler.db, buf)
		return true
	case "scrape":
		if enabled, _ := config.GetBool("scrape", true); !enabled {
			return false
		}

		scrape(r.URL.RawQuery, user, handler.db, buf)

		return true
	case "metrics":
		metrics(r.Header.Get("Authorization"), handler.db, buf)
		return true
	}

	return false
}

func (handler *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler.terminate {
		return
	}

	handler.waitGroup.Add(1)
	buf := handler.bufferPool.Take()

	defer w.(http.Flusher).Flush()
	defer handler.waitGroup.Done()
	defer handler.bufferPool.Give(buf)

	defer func() {
		if err := recover(); err != nil {
			log.Error.Printf("ServeHTTP panic - %v\nURL was: %s", err, r.URL)
			log.WriteStack()

			// If present then writing response failed and we should not attempt to write again
			if w.Header().Get("Content-Type") == "" {
				buf.Reset()
				w.WriteHeader(http.StatusInternalServerError)
			}

			collectors.IncrementErroredRequests()
		}
	}()

	exists, status := handler.respond(r, buf), http.StatusOK
	if !exists {
		status = http.StatusNotFound
	}

	w.Header().Add("Content-Length", strconv.Itoa(buf.Len()))
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(status)

	if _, err := w.Write(buf.Bytes()); err != nil {
		panic(err)
	}

	atomic.AddUint64(&handler.requests, 1)
}

func Start() {
	var err error

	handler = &httpHandler{db: &database.Database{}, startTime: time.Now()}

	bufferPool := util.NewBufferPool(500, 500)
	handler.bufferPool = bufferPool

	addr, _ := config.Section("http").Get("addr", ":34000")
	readTimeout, _ := config.Section("http").GetInt("read_timeout", 2)
	writeTimeout, _ := config.Section("http").GetInt("write_timeout", 2)

	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
	}

	handler.db.Init()
	record.Init()

	handler.normalRegisterer = prometheus.NewRegistry()
	handler.normalCollector = collectors.NewNormalCollector()
	handler.normalRegisterer.MustRegister(handler.normalCollector)

	// Register additional metrics for DefaultGatherer
	handler.adminCollector = collectors.NewAdminCollector()
	prometheus.MustRegister(handler.adminCollector)

	listener, err = net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	/*
	 * Behind the scenes, this works by spawning a new goroutine for each client.
	 * This is pretty fast and scalable since goroutines are nice and efficient.
	 */
	log.Info.Printf("Ready and accepting new connections on %s", addr)

	_ = server.Serve(listener)

	// Wait for active connections to finish processing
	handler.waitGroup.Wait()

	_ = server.Close() // close server so that it does not Accept(), https://github.com/golang/go/issues/10527

	log.Info.Println("Now closed and not accepting any new connections")

	handler.db.Terminate()

	log.Info.Println("Shutdown complete")
}

func Stop() {
	// Closing the listener stops accepting connections and causes Serve to return
	_ = listener.Close()
	handler.terminate = true
}

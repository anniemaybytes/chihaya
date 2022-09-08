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
	"context"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"chihaya/collectors"
	"chihaya/config"
	"chihaya/database"
	"chihaya/log"
	"chihaya/record"
	"chihaya/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/zeebo/bencode"
)

type httpHandler struct {
	startTime time.Time

	normalRegisterer prometheus.Registerer
	normalCollector  *collectors.NormalCollector
	adminCollector   *collectors.AdminCollector

	bufferPool *util.BufferPool

	db *database.Database

	requests atomic.Uint64

	contextTimeout time.Duration

	waitGroup sync.WaitGroup
	terminate bool
}

var (
	handler  *httpHandler
	listener net.Listener
)

func failure(err string, buf *bytes.Buffer, interval time.Duration) {
	failureData := make(map[string]interface{})
	failureData["failure reason"] = err
	failureData["interval"] = interval / time.Second     // Assuming in seconds
	failureData["min interval"] = interval / time.Second // Assuming in seconds

	data, errz := bencode.EncodeBytes(failureData)
	if errz != nil {
		panic(errz)
	}

	buf.Reset()

	if _, errz = buf.Write(data); errz != nil {
		panic(errz)
	}
}

func (handler *httpHandler) respond(r *http.Request, buf *bytes.Buffer) int {
	dir, action := path.Split(r.URL.Path)
	if action == "" {
		return http.StatusNotFound
	}

	/*
	 * ===================================================
	 * Handle public endpoints (/:action)
	 * ===================================================
	 */

	passkey := path.Dir(dir)[1:]
	if passkey == "" {
		switch action {
		case "alive":
			return alive(buf)
		}

		return http.StatusNotFound
	}

	/*
	 * ===================================================
	 * Handle private endpoints (/:passkey/:action)
	 * ===================================================
	 */

	user, err := isPasskeyValid(r.Context(), passkey, handler.db)
	if err != nil {
		return http.StatusRequestTimeout
	} else if user == nil {
		failure("Your passkey is invalid", buf, 1*time.Hour)
		return http.StatusOK
	}

	switch action {
	case "announce":
		return announce(r.Context(), r.URL.RawQuery, r.Header, r.RemoteAddr, user, handler.db, buf)
	case "scrape":
		if enabled, _ := config.GetBool("scrape", true); !enabled {
			return http.StatusNotFound
		}

		return scrape(r.Context(), r.URL.RawQuery, user, handler.db, buf)
	case "metrics":
		return metrics(r.Context(), r.Header.Get("Authorization"), handler.db, buf)
	}

	return http.StatusNotFound
}

func (handler *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler.terminate {
		return
	}

	// Count new request (done before everything else so that failed/timeout numbers match)
	handler.requests.Add(1)

	// Mark request as being handled, so that server won't shutdown before we're done with it
	handler.waitGroup.Add(1)
	defer handler.waitGroup.Done()

	// Prepare, consume and mark as being used buffer for request
	var (
		buf      *bytes.Buffer
		bufInUse sync.WaitGroup
	)

	buf = handler.bufferPool.Take()

	bufInUse.Add(1)
	defer bufInUse.Done()

	// Spawn new goroutine that waits for buffer to be unused before returning it back to pool for reuse
	go func() {
		bufInUse.Wait()
		handler.bufferPool.Give(buf)
	}()

	// Flush buffered data to client
	defer w.(http.Flusher).Flush()

	// Gracefully handle panics so that they're confined to single request and don't crash server
	defer func() {
		if err := recover(); err != nil {
			log.Error.Printf("Recovered from panicking request handler - %v\nURL was: %s", err, r.URL)
			log.WriteStack()

			collectors.IncrementErroredRequests()

			if w.Header().Get("Content-Type") == "" {
				buf.Reset()
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}()

	// Prepare and start new context with timeoud to abort long-running requests
	var (
		ctx    context.Context
		cancel context.CancelFunc
		status int
		done   = make(chan struct{})
		errz   = make(chan any, 1)
	)

	ctx, cancel = context.WithTimeout(r.Context(), handler.contextTimeout)
	defer cancel()

	/* Mark buffer as in use; we can't recycle buffer until both ServeHTTP and handler are done with it.
	This must be done outside of goroutine itself to avoid concurrency issues where select statement below
	finishes before our handler goroutine has a chance to start, bringing WaitGroup counter to 0 while
	other goroutine is Waiting, which is invalid scenario and will result in panic. */
	bufInUse.Add(1)

	// Start new blocking handler in goroutine
	go func() {
		defer func() {
			if err := recover(); err != nil {
				errz <- err
			}
		}()

		// Once this goroutine exits, we should free its hold on buffer
		defer bufInUse.Done()

		/* Pass flow to handler; note that handler should be responsible for actually cancelling
		its own work based on request context cancellation */
		status = handler.respond(r, buf)

		// Close channel, marking that handler finished its work
		close(done)
	}()

	// Await on either panic, handler to finish or context deadline to expire
	select {
	case err := <-errz:
		panic(err)
	case <-done:
		w.Header().Add("Content-Length", strconv.Itoa(buf.Len()))
		w.Header().Add("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write(buf.Bytes())
	case <-ctx.Done():
		collectors.IncrementTimeoutRequests()

		switch err := ctx.Err(); err {
		case context.DeadlineExceeded:
			failure("Request context deadline exceeded", buf, 5*time.Minute)

			w.Header().Add("Content-Length", strconv.Itoa(buf.Len()))
			w.Header().Add("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK) // Required by torrent clients to interpret failure response
			_, _ = w.Write(buf.Bytes())
		default:
			w.WriteHeader(http.StatusRequestTimeout)
		}
	}
}

func Start() {
	handler = &httpHandler{db: &database.Database{}, startTime: time.Now()}

	/* Initialize reusable buffer pool; this is faster than allocating new memory for every request.
	If necessary, new memory will be allocated when pool is empty, however. */
	bufferPool := util.NewBufferPool(500, 512)
	handler.bufferPool = bufferPool

	addr, _ := config.Section("http").Get("addr", ":34000")
	readTimeout, _ := config.Section("http").Section("timeout").GetInt("read", 1)
	readHeaderTimeout, _ := config.Section("http").Section("timeout").GetInt("read_header", 2)
	writeTimeout, _ := config.Section("http").Section("timeout").GetInt("write", 3)
	idleTimeout, _ := config.Section("http").Section("timeout").GetInt("idle", 30)

	// Set appropriate context timeout based on write timeout; 200ms should be enough as a wiggle room
	handler.contextTimeout = time.Duration(writeTimeout)*time.Second - 200*time.Millisecond

	// Create new server instance
	server := &http.Server{
		Handler:           handler,
		ReadTimeout:       time.Duration(readTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(readHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(writeTimeout) * time.Second,
		IdleTimeout:       time.Duration(idleTimeout) * time.Second,
	}

	if idleTimeout <= 0 {
		log.Warning.Print("Setting idleTimeout <= 0 disables Keep-Alive which might negatively impact performance")
		server.SetKeepAlivesEnabled(false)
	}

	// Initialize database and recorder
	handler.db.Init()
	record.Init()

	// Register default prometheus collector
	handler.normalRegisterer = prometheus.NewRegistry()
	handler.normalCollector = collectors.NewNormalCollector()
	handler.normalRegisterer.MustRegister(handler.normalCollector)

	// Register additional metrics for DefaultGatherer
	handler.adminCollector = collectors.NewAdminCollector()
	prometheus.MustRegister(handler.adminCollector)

	// Start TCP listener
	var err error

	listener, err = net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	log.Info.Printf("Ready and accepting new connections on %s", addr)

	/* Start serving new request. Behind the scenes, this works by spawning a new goroutine for each client.
	This is pretty fast and scalable since goroutines are nice and efficient. Blocks until TCP listener is closed. */
	_ = server.Serve(listener)

	// Wait for active connections to finish processing
	handler.waitGroup.Wait()

	// Close server so that it does not Accept(), https://github.com/golang/go/issues/10527
	_ = server.Close()

	log.Info.Print("Now closed and not accepting any new connections")

	// Close database connection
	handler.db.Terminate()

	log.Info.Print("Shutdown complete")
}

func Stop() {
	// Closing the listener stops accepting connections and causes Serve to return
	_ = listener.Close()
	handler.terminate = true
}

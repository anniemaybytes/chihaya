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
	"log/slog"
	"net"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"chihaya/collector"
	"chihaya/config"
	"chihaya/database"
	"chihaya/util"

	"github.com/valyala/fasthttp"
)

type httpHandler struct {
	startTime time.Time

	bufferPool *util.BufferPool

	db *database.Database

	requests atomic.Uint64

	waitGroup sync.WaitGroup
	terminate bool
}

var (
	handler  *httpHandler
	listener net.Listener
)

func (handler *httpHandler) serve(ctx *fasthttp.RequestCtx) {
	if handler.terminate {
		return
	}

	// Count new request (done before everything else so that failed/timeout numbers match)
	handler.requests.Add(1)
	collector.IncrementRequests()

	// Mark request as being handled, so that server won't shutdown before we're done with it
	handler.waitGroup.Add(1)
	defer handler.waitGroup.Done()

	// Take buffer from pool and mark buf to be returned after we are done with it
	buf := handler.bufferPool.Take()
	defer handler.bufferPool.Give(buf)

	// Gracefully handle panics so that they're confined to single request and don't crash server
	defer func() {
		if err := recover(); err != nil {
			slog.Error("recovered from panicking request handler", "err", err, "url", ctx.URI())

			collector.IncrementErroredRequests()

			if len(ctx.Response.Header.ContentType()) == 0 {
				buf.Reset()
				ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
			}
		}
	}()

	/* Pass flow to handler; note that handler should be responsible for actually canceling
	its own work based on request context cancellation */
	status := func() int {
		dir, file := path.Split(string(ctx.URI().Path()))

		switch dir {
		case "/":
			switch file {
			case "alive":
				return alive(ctx, handler.db, buf)
			case "metrics":
				if enabled, _ := config.GetBool("enable_metrics", false); !enabled {
					return fasthttp.StatusNotFound
				}

				return metrics(ctx, handler.db, buf)
			}
		default:
			user := isPasskeyValid(path.Base(dir), handler.db)
			if user == nil {
				failure("Your passkey is invalid", buf, 1*time.Hour)
				return fasthttp.StatusOK
			}

			ctx.SetUserValue("user", user) // Pass user in request's context

			switch file {
			case "announce":
				return announce(ctx, user, handler.db, buf)
			case "scrape":
				if enabled, _ := config.GetBool("enable_scrape", true); !enabled {
					return fasthttp.StatusNotFound
				}

				return scrape(ctx, user, handler.db, buf)
			}
		}

		return fasthttp.StatusNotFound
	}()

	ctx.Response.Header.SetContentLength(buf.Len())
	ctx.Response.Header.SetContentTypeBytes([]byte("text/plain"))
	ctx.Response.SetStatusCode(status)
	_, _ = buf.WriteTo(ctx)
}

func (handler *httpHandler) error(ctx *fasthttp.RequestCtx, err error) {
	ctx.Response.ResetBody()
	ctx.Response.Header.SetContentLength(0)
	ctx.Response.Header.SetContentTypeBytes([]byte("text/plain"))

	//goland:noinspection GoTypeAssertionOnErrors
	if _, ok := err.(*fasthttp.ErrSmallBuffer); ok {
		ctx.Response.SetStatusCode(fasthttp.StatusRequestHeaderFieldsTooLarge)
		return
	} else if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
		ctx.Response.SetStatusCode(fasthttp.StatusRequestTimeout)
		return
	}

	ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
}

func Start() {
	handler = &httpHandler{db: &database.Database{}, startTime: time.Now()}

	/* Initialize reusable buffer pool; this is faster than allocating new memory for every request.
	If necessary, new memory will be allocated when pool is empty, however. */
	bufferPool := util.NewBufferPool(512)
	handler.bufferPool = bufferPool

	addr, _ := config.Section("http").Get("addr", ":34000")
	readTimeout, _ := config.Section("http").Section("timeout").GetInt("read", 300)
	writeTimeout, _ := config.Section("http").Section("timeout").GetInt("write", 500)
	idleTimeout, _ := config.Section("http").Section("timeout").GetInt("idle", 30)

	// Create new server instance
	server := &fasthttp.Server{
		Handler:                      handler.serve,
		ErrorHandler:                 handler.error,
		ReadTimeout:                  time.Duration(readTimeout) * time.Millisecond,
		WriteTimeout:                 time.Duration(writeTimeout) * time.Millisecond,
		IdleTimeout:                  time.Duration(idleTimeout) * time.Second,
		GetOnly:                      true,
		DisablePreParseMultipartForm: true,
		NoDefaultServerHeader:        true,
		NoDefaultDate:                true,
		NoDefaultContentType:         true,
		CloseOnShutdown:              true,
	}

	if idleTimeout <= 0 {
		server.DisableKeepalive = true
	}

	// Start new goroutine to calculate throughput
	go func() {
		prevTime := time.Now()
		prevRequests := handler.requests.Load()

		for !handler.terminate {
			time.Sleep(time.Minute)

			duration := time.Since(prevTime)
			deltaRequests := handler.requests.Load() - prevRequests

			throughput := int(float64(deltaRequests)/duration.Seconds()*60 + 0.5)
			collector.UpdateThroughput(throughput)

			prevTime = time.Now()
			prevRequests = handler.requests.Load()

			slog.Info("current throughput", "rpm", throughput, "duration", duration, "delta", deltaRequests)
		}
	}()

	// Initialize database
	handler.db.Init()

	// Start TCP listener
	var err error

	listener, err = net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	slog.Info("ready and accepting new connections", "addr", addr)

	/* Start serving new request. Behind the scenes, this works by spawning a new goroutine for each client.
	This is pretty fast and scalable since goroutines are nice and efficient. Blocks until TCP listener is closed. */
	_ = server.Serve(listener)

	// Wait for active connections to finish processing
	handler.waitGroup.Wait()

	_ = server.Shutdown()

	slog.Info("now closed and not accepting any new connections")

	// Close database connection
	handler.db.Terminate()

	slog.Info("shutdown complete")
}

func Stop() {
	if listener != nil {
		// Closing the listener stops accepting connections and causes Serve to return
		_ = listener.Close()
	}

	handler.terminate = true
}

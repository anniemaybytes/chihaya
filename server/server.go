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
	"chihaya/config"
	cdb "chihaya/database"
	"chihaya/record"
	"chihaya/util"
	"log"
	"net"
	"net/http"
	"net/url"
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
	db               *cdb.Database
	normalRegisterer prometheus.Registerer
	normalCollector  *NormalCollector

	startTime time.Time
}

type queryParams struct {
	params     map[string]string
	infoHashes []string
}

var privateIPBlocks []*net.IPNet

func (p *queryParams) get(which string) (ret string, exists bool) {
	ret, exists = p.params[which]
	return
}

func (p *queryParams) getUint64(which string) (ret uint64, exists bool) {
	str, exists := p.params[which]
	if exists {
		var err error

		exists = false

		ret, err = strconv.ParseUint(str, 10, 64)
		if err == nil {
			exists = true
		}
	}

	return
}

func getPublicIpV4(ipAddr string, exists bool) (string, bool) {
	if !exists { // Already does not exist, fail
		return ipAddr, exists
	}

	ip := net.ParseIP(ipAddr)
	if ip == nil { // Invalid IP provided, fail
		return ipAddr, false
	}

	if ip.To4() == nil { // IPv6 provided, fail
		return ipAddr, false
	}

	private := false

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		private = true
	} else {
		for _, block := range privateIPBlocks {
			if block.Contains(ip) {
				private = true
				break
			}
		}
	}

	return ipAddr, !private
}

func failure(err string, buf *bytes.Buffer, interval time.Duration) {
	failureData := make(map[string]interface{})
	failureData["failure reason"] = err
	failureData["interval"] = interval / time.Second     // Assuming in seconds
	failureData["min interval"] = interval / time.Second // Assuming in seconds

	data, errz := bencode.EncodeBytes(failureData)
	if errz != nil {
		panic(err)
	}

	buf.Write(data)
}

/*
 * URL.Query() is rather slow, so I rewrote it
 * Since the only parameter that can have multiple values is info_hash for scrapes, handle this specifically
 */
func (handler *httpHandler) parseQuery(query string) (ret *queryParams, err error) {
	ret = &queryParams{make(map[string]string), nil}
	queryLen := len(query)

	onKey := true

	var keyStart int

	var keyEnd int

	var valStart int

	var valEnd int

	hasInfoHash := false

	var firstInfoHash string

	for i := 0; i < queryLen; i++ {
		separator := query[i] == '&' || query[i] == ';'
		if separator || i == queryLen-1 { // ';' is a valid separator as per W3C spec
			if onKey {
				keyStart = i + 1
				continue
			}

			if i == queryLen-1 && !separator {
				if query[i] == '=' {
					continue
				}

				valEnd = i
			}

			keyStr, err1 := url.QueryUnescape(query[keyStart : keyEnd+1])
			if err1 != nil {
				err = err1
				return
			}

			valStr, err1 := url.QueryUnescape(query[valStart : valEnd+1])
			if err1 != nil {
				err = err1
				return
			}

			ret.params[keyStr] = valStr

			if keyStr == "info_hash" {
				if hasInfoHash {
					// There is more than one info_hash
					if ret.infoHashes == nil {
						ret.infoHashes = []string{firstInfoHash}
					}

					ret.infoHashes = append(ret.infoHashes, valStr)
				} else {
					firstInfoHash = valStr
					hasInfoHash = true
				}
			}

			onKey = true
			keyStart = i + 1
		} else if query[i] == '=' {
			onKey = false
			valStart = i + 1
		} else if onKey {
			keyEnd = i
		} else {
			valEnd = i
		}
	}

	return
}

func (handler *httpHandler) respond(r *http.Request, buf *bytes.Buffer) {
	dir, action := path.Split(r.URL.Path)
	if len(dir) != 34 {
		failure("Malformed request - missing passkey", buf, 1*time.Hour)
		return
	}

	passkey := dir[1:33]

	params, err := handler.parseQuery(r.URL.RawQuery)

	if err != nil {
		failure("Error parsing query", buf, 1*time.Hour)
		return
	}

	handler.db.UsersMutex.RLock()
	user, exists := handler.db.Users[passkey]
	handler.db.UsersMutex.RUnlock()

	if !exists {
		failure("Your passkey is invalid", buf, 1*time.Hour)
		return
	}

	ipAddr, exists := func() (string, bool) {
		ipV4, existsV4 := getPublicIpV4(params.get("ipv4")) // first try to get ipv4 address if client sent it
		ip, exists := getPublicIpV4(params.get("ip"))       // then try to get public ip if sent by client

		if existsV4 && exists && ip != ipV4 { // fail if ip and ipv4 are not same, and both are provided
			return "", false
		}

		if existsV4 {
			return ipV4, true
		}

		if exists {
			return ip, true
		}

		// check if there is proxy in header IF allowed in config
		proxy_header_type := config.Get("proxy")
		if proxy_header_type != "" {
			ips, exists := r.Header[proxy_header_type]
			if exists && len(ips) > 0 {
				return ips[0], true
			}
		}

		// check for IP in socket
		portIndex := len(r.RemoteAddr) - 1
		for ; portIndex >= 0; portIndex-- {
			if r.RemoteAddr[portIndex] == ':' {
				break
			}
		}

		if portIndex != -1 {
			return r.RemoteAddr[0:portIndex], true
		}

		return "", false // everything failed, abort request
	}()

	if !exists {
		failure("Failed to parse IP address", buf, 1*time.Hour)
		return
	}

	ipBytes := (net.ParseIP(ipAddr)).To4()
	if nil == ipBytes {
		failure("Assertion failed (net.ParseIP(ipAddr)).To4() == nil)! please report this issue to staff", buf, 1*time.Hour)
		return
	}

	switch action {
	case "announce":
		announce(params, user, ipAddr, handler.db, buf)
		return
	case "scrape":
		scrape(params, handler.db, buf)
		return
	case "metrics":
		metrics(r.Header.Get("Authorization"), handler.normalCollector, handler.db, buf)
		return
	}

	failure("Unknown action", buf, 1*time.Hour)
}

var handler *httpHandler
var listener net.Listener

func (handler *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler.terminate {
		return
	}

	handler.waitGroup.Add(1)
	defer handler.waitGroup.Done()

	defer func() {
		err := recover()
		if err != nil {
			log.Printf("!!! ServeHTTP panic !!! %v", err)
		}
	}()

	buf := handler.bufferPool.Take()
	defer handler.bufferPool.Give(buf)

	handler.respond(r, buf)

	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Content-Length", strconv.Itoa(buf.Len()))

	// The response should always be 200, even on failure
	_, _ = w.Write(buf.Bytes())

	atomic.AddUint64(&handler.requests, 1)

	w.(http.Flusher).Flush()
}

func Start() {
	var err error

	for _, cidr := range []string{
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"100.64.0.0/10",  // RFC6598
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Printf("!!! CRITICAL !!! parse error on %q: %v", cidr, err)
		} else {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}

	handler = &httpHandler{db: &cdb.Database{}, startTime: time.Now()}

	bufferPool := util.NewBufferPool(500, 500)
	handler.bufferPool = bufferPool

	server := &http.Server{
		Handler:     handler,
		ReadTimeout: 20 * time.Second,
	}

	handler.db.Init()
	record.Init()

	handler.normalRegisterer = prometheus.NewRegistry()
	handler.normalCollector = NewNormalCollector()
	handler.normalRegisterer.MustRegister(handler.normalCollector)

	listener, err = net.Listen("tcp", config.Get("addr"))
	if err != nil {
		panic(err)
	}

	/*
	 * Behind the scenes, this works by spawning a new goroutine for each client.
	 * This is pretty fast and scalable since goroutines are nice and efficient.
	 */
	log.Printf("Ready and accepting new connections on %s", config.Get("addr"))

	_ = server.Serve(listener)

	// Wait for active connections to finish processing
	handler.waitGroup.Wait()

	_ = server.Close() // close server so that it does not Accept(), https://github.com/golang/go/issues/10527

	log.Println("Now closed and not accepting any new connections")

	handler.db.Terminate()

	log.Println("Shutdown complete")
}

func Stop() {
	// Closing the listener stops accepting connections and causes Serve to return
	_ = listener.Close()
	handler.terminate = true
}

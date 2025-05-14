# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

## v12.0.0
### Added
- New metric `chihaya_purge_inactive_peers_seconds`

### Fixed
- Preserve original snatched time in case of multiple snatches
- Various typos
- Do not substract 1 from peer count when leeching

### Changed
- Bump minimum Go version to 1.24
- Avoid loading torrents in scrape if no info hashes were provided
- Do not send "min interval" on failure
- Replace usage of `UnsafeRand` with `UnsafeIntn`
- Update failure messages in few places
- Do not consume users channel for empty actions

### Removed
- Support for old-style BEP7 multi-homed connections in form of `&ipv4=` and `&ipv6=` query string params
- Support for "interval" and "min interval" in scrape as no known client supports reading these
- Public-facing metrics and custom metrics authorization (`/metrics` endpoint protection is now responsibility of operator)
- Custom error handling on `/alive` endpoint

## v11.0.0
### Fixed
- Fix panic when attempting to shutdown chihaya before main server loop has started
- Do not allocate new bytes buffer for each bencode operation
- Fix `TestSerializer` not failing when serialization errors out
- Fix potential race condition for global variable in `record.Record`

### Changed
- Refactor cc utility to use `io.Writer` and `io.Reader`
- Improve Recorder section of README
- Use `sync.Pool` as backend instead of channels for `util/BufferPool`
- Replace runtime CPU profiler with `net/http/pprof` debug server
- Make `HitAndRuns`, `TorrentGroupFreeleech`, `Clients` and `GlobalFreeleech` use `atomic` primitives
- Use `[]byte` instead of string to prevent lookups in heap or via golang internal `aeshash`
- Move closing of flush channels to separate function
- Replace gob serialization with custom binary marshalling
- Change `Peer.Addr` to fixed array
- Re-allocate empty maps for Leechers to reduce memory usage
- Optimize Peer in-memory size
- Cleanup bencode utility
- Rename `database/Record*` -> `database/Queue*`
- Simplify request handling by processing them directly, and pass contextTimeout to handler response
- Make Queue methods optional async, with fast pathway without goroutine spawn
- Update schema.sql for `STRICT_TRANS_TABLES`
- Replace `Users`/`Torrents` locks with `atomic.Pointer`
- Replace `net/http` with https://github.com/valyala/fasthttp
- Refactor query parsing into a struct via `fasthttp`
- Do not spawn new goroutine for `QueueSnatch`
- Keep single sql.DB instead of multiple custom Connection
- Remove usage of `TEMPORARY` tables for flushes
- Alter config to take database DSN directly
- Make recorder output `csv` instead of `json`

### Added
- Introduce and use non-crypto Rand for tests and announce interval drift
- Enable mutex and blocks profiling when running with -P flag
- Introduce anonymize command to anonymize binary cache dumps
- Enable PGO builds for chihaya
- Separate metrics for `context.DeadlineExceeded` and `context.Cancelled`
- Add test case for `server/util.failure`

## v10.5.0
### Changed
- Adjust histogram buckets to better capture wider range of data

## v10.4.1
### Fixed
- Load group freeleech during initialization

## v10.4.0
### Changed
- Separate torrents and group freeleech reload
- Reload approved client list on every refresh

## v10.3.0
### Added
- Prometheus metric `throughput`

## v10.2.0
### Changed
- Increase announce interval for non-existing torrents

## v10.1.0
### Changed
- Use new `util.Semaphore` (introduced in `v10.0.0`) in `database/flush`

## v10.0.0
### Added
- Use `context.WithTimeout` to cancel long-running request

### Changed
- Exit on uninitialized `Record` error
- Improve and standardize logging across various files
- Update warning messages in `readConfig`
- Make serialization atomic by using `os.Rename`
- Refactor old `/check` into new `/alive` endpoint
- Bump minimum Go version to 1.19

## v9.0.1
### Fixed
- Record of ID 0 in `approved_clients` table being treated as non-approved
- Requests should be counted for Prometheus collector before handling them in `ServerHTTP`
- Handle `TorrentsMutex`'s `RUnlock` in defer in `server/scrape.go`
- Incorrect usage of `log.Panic` in `record/record.go`

## v9.0.0
### Added
- Allow configuring `ReadHeaderTimeout`, `IdleTimeout` and `SetKeepAlivesEnabled`

### Changed
- Refactor config file structure:
    - Move `read_timeout`, `write_timeout` to new `http.timeout` section and remove
      `_timeout` suffix

## v8.1.0
### Fixed
- Avoid 'superfluous response.WriteHeader call' error when handling panic in `ServeHTTP`

### Changed
- Do not set GOMAXPROCS in chihaya/main as it defaults to system CPUs on new Go versions already

## v8.0.0
### Added
- Ability to control HTTP timeouts via `read_timeout` and `write_timeout`
(in new `http` config section)

### Changed
- Refactor config file structure:
    - Move `strict_port`, `numwant` and `maxnumwant` to new `announce` section
    - Move `addr`, `admin_token` to new `http` section
    - Rename `proxy` to `proxy_header` and move it to `http` section

### Removed
- `flush_groups` configuration option and related code

## v7.1.0
### Changed
- Modify HTTP read/write timeouts

## v7.0.0
### Removed
- Support for semicolons in query parameters (https://golang.org/doc/go1.17#semicolons)

### Changed
- Due to above change, bump minimum Go version to 1.17

## v6.1.0
### Changed
- Load freeleech data for torrent groups separately from torrents themselves

## v6.0.0
### Changed
- Switch build system to `Makefile`

## v5.3.0
### Fixed
- Null pointer dereference on SQL rows in `purgeInactivePeers` and `load*` functions
- Wrong max deadlock count being printed in warning message

### Changed
- Improve database tests

## v5.2.0
### Added
- Database tests using `go-testfixtures/testfixtures` were added

## v5.1.1
### Fixed
- Download and upload multiplier being switched around when loading new user

## v5.1.0
### Added
- `flush_groups` config option was added to control whether groups
should be updated whenever related torrent is flushed

## v5.0.0
### Changed
- Renamed table `client_whitelist` to `approved_clients`
- Renamed prometheus stat `chihaya_whitelist` to `chihaya_clients`
- Removed all references to whitelist/blacklist

## v4.2.0
### Changed
- Use `crypto/rand` in `util` for `RandStringBytes` and `Rand`

## v4.1.0
### Added
- `chihaya_sql_errors_count` metric now tracks SQL errors

### Changed
- Temporary tables for user and torrent flushes are now used more efficiently

## v4.0.0
### Changed
- Do not use enum for `transfer_history`

## v3.5.0
### Added
- Update the timestamp on torrents group when flushing torrents

## v3.4.0
### Changed
- Do not return scrape information for torrents that user can not download

## v3.3.0
### Added
- New format for `record` including more useful data

## v3.2.0
### Added
- Prometheus metric for tracking aborted deadlocks
- Tests for `GetInt` and `GetBool` in `config`
- Tests for `server/params`

### Changed
- Clarify `Your client is not approved` message by using `peer_id` instead of `id`

## v3.1.1
### Fixed
- Remove unnecessary quoting of integer columns for database queries

## v3.1.0
### Added
- Ability to configure default/maximum `numwant` from config

### Fixed
- `ServeHTTP` panicking when query string included empty parameter followed by delimiter 
(`?bug=&yes=`)

### Changed
- Ensure `peer_id` is always 20 bytes
- Rewrite query string parser
- Ignore `info_hash` if it isn't exactly 20 bytes

## v3.0.0
### Added
- Log failing URL on panic in `ServeHTTP`
- `/check` endpoint for healt-checking
- Ability to configure intervals from `config.json`
- Ability to configure deadlock behavior from `config.json`
- Ability to configure channels buffer length from `config.json`
- Ability to `restore` cache dump from JSON

### Fixed
- `successful` typo
- Handle errors early for prepared statements

### Changed
- Moved `InitPrivateIPBlocks` to `init` inside `server.go`
- Panic in metrics instead of returning empty body on error
- Golint reformat code
- Narrow `bytes.Buffer` in server to `io.Writer`
- Eliminate long lines
- golangci: disable default linters and specify hard list of enabled ones
- Return HTTP 500 when panicking in `ServeHTTP`
- Use `time.Duration.String` for time formatting across code
- Remove `prepareStatement` from `database.go`
- database: `OpenDatabaseConnection` -> `Open`
- database: `exec` -> `execute`, `execBuffer` -> `exec` (follows PDO)
- Reformat README with better explanation of config file
- Count only unique deadlocks for `chihaya_deadlock_count`
- Move query parsing logic to separate module
- Do not parse query string globally in server, have each action handle it separately if needed
- Completely rework passkey and action logic allowing for non-passkey protected endpoints
- Return 404 with empty body when request is malformed or action is not recognized
instead of bencoded message
- Handle errors of `w.Write()` in `ServeHTTP`
- Various code cleanups

## v2.5.0
### Added
- Ability to enable or disable strict port checking in `config.json`
- Add bencode utility for converting between bencode and JSON
- Help screen for `chihaya`
- Ability to configure inactive peer time and announce drift

### Fixed
- Call to `panic()` with wrong argument in `failure()` (`server.go`)
- `cc` utility was overwriting torrent cache file with JSON version
- Header configured in `proxy` was not being respected
- `@config/GetInt()` was improperly implemented using `int` instead of `json.Number`
- `InactiveAnnounceInterval` did not account for configured announce drift properly
- `config` Getters were failing on unexpected values (such as `nil`) instead of falling back 
to default
- New flush logic for users and torrents was not properly utilizing temporary table leading to 
data loss in cases where single user or torrent was present multiple times in a flushed channel

### Changed
- Have `Get` functions in `config` return `bool` indicating whether default was used
- Rename `cache-dump` to `cc`
- Do not use `chihaya/log` inside `cc` utility
- Split types from `database` into `types` sub-package
- Move logging of Info from stderr to stdout
- Rename `profile` flag to `P`
- Make announce drift random
- Make @config/GetInt() return `int` instead of `int64`
- `config` now reads numeric values from JSON as numbers instead of floats

## v2.4.0
### Added
- Ability to configure logging of flushes in `config.json`
- Utility for exporting cache into readable JSON format
- Ability to specify custom arguments `CGO_ENABLED`, `GOOS` and `GOARCH` in Dockerfile 
during build process
- Dump stacktrace when error is encountered

### Fixed
- README was improperly showing `null` as valid value for `admin_token` and `proxy`.

### Changed
- Changed `users` and `torrents` flush SQL queries to use temporary table with `UPDATE` 
instead of `INSERT INTO ... ON DUPLICATE KEY ...`, this avoids rare cases where previously 
removed entry from these tables is inserted back with default values by chihaya
- Code cleanup
- Make torrentId `uint32`
- Moved `main.go` to `cmd/chihaya` to allow for building multiple binaries

## v2.3.0
### Added
- Support for interval time in `/scrape` endpoint
- Support for `failure reason` in `/scrape` endpoint
- Add port to `transfer_ips`
- Add port to JSON recorder

### Changed
- Use Alpine docker image as builder
- Better formatting of warning messages in config.go
- Split mandatory parameters check into separate sections with their own failure messages
- Do not lock TorrentsMutex if no info_hash is provided to `/scrape`
- Print name of unsupported action in failure message
- Refactor logging across codebase
- Record hidden ip as 2130706433 (127.0.0.1) instead of 0 (0.0.0.0)
- Refactor config allowing for default values to be given in code
- Move `record` and `scrape` from build tags into config variables

## v2.2.0
### Added
- Additional metrics in prometheus for whitelist, hit and runs
- Additional metrics in prometheus for channels, reload time, flush time, deadlock
- Validate port provided by client (must be between 1024 and 65535)

### Changed
- Code cleanup
- Move IP check from server to announce

## v2.1.0
### Added
- Ability to control which (if any) header is used for proxy support

### Changed
- Fail request if IP provided by client is in private range

## v2.0.0
### Added
- Support for prometheus metrics

### Changed
- Run docker container as UID 1000
- Use Go 1.14

### Removed
- Plaintext `/stats` endpoint

## v1.3.1
### Added
- Print number of CPUs when started with `profile` flag

## v1.3.0
### Added
- Dockerfile

### Changed
- Code style fixes

## v1.2.0
### Changed
- Optimize structs memory alignment
- Code cleanup

## v1.0.0
### Changed
- Rewrote database using database/sql and MySQL driver

## v0.8.0
### Added
- Tests for config.go

### Changed
- Only one peer per user is sent when seeders requested

## v0.7.0
### Added
- Tests for record.go
- Tests for util.go
- Print peerId on 'client not approved' failure message
- RandStringBytes in util
- Tests for util/bufferpool

### Fixed
- Broken error handling for parseQuery in server.go
- Explicitly Close() server itself on shutdown (https://github.com/golang/go/issues/10527)

### Changed
- Migrate to bencode library (https://github.com/zeebo/bencode)
- Explicitly ignore error handling on shutdown
- Code cleanup
- Remove obsolete TODOs
- Simplify ipAddr handling in announce
- Export ConfigMap from config
- Extract record into separate package
- Synchronously populate initial data from database into memory

## v0.6.0
### Added
- Support for archived whitelist entries
- Failsafe to ensure all IPs are always IPv4 only
- Allow to build without `scrape` and `recorder` via build tags

### Fixed
- Ensure down/up multipliers are always positive
- Make `transferHistory` wait in `purgeInactivePeers` atomic with add in 
`flushTransferHistory` 
- Ensure `transferHistoryWaitGroup` is released properly on loop break when 
`transferHistoryChannel` is empty
- Do not break main loop in `flushTransferHistory` when channel is empty

### Changed
- Rename whitelist table from `xbt_whitelist` to `client_whitelist`
- Update torrent's `last_action` only if announced action is seeding
- Increase AnnounceInterval, MinAnnounceInterval and PurgeInactiveInterval
- Update to MariaDB 10.3.3 syntax (https://mariadb.com/kb/en/library/values-value)
- Move external modules to `go.mod`
- Check error value from `.Encode` during serialization
- Remove `connectable` from `transfer_history` flush
- Code cleanup and formatting

## v0.5.0
### Added
- Support for disabling logging of IP for user
- Support for `no_peer_id`
- Support for `ipv4` query string
- Support for interval in `failure`

### Changed
- Ignore `paused` event (https://www.bittorrent.org/beps/bep_0021.html)
- Cleanup unused code
- Have `interval` for announce include small drift in form of `min(600, seeders)`
- Default to compact responses unless explicitly asked for by `compact=0`
- Bump default `numWant` from 20 to 25

## v0.4.0
### Added
- Add LICENSE
- Simple JSON announce recorder
- Handle SIGTERM in addition to INT for graceful restarts
- Support for group based freeleech
- Allow downloading Hit and Runs even if `DisableDownload` is set for user

### Fixed
- Returning 50 peers when the client asks for 0
- Fix peer inactivity query
- Broken bencode on `/scrape`
- Swapped up/down multiplier for initial torrent load
- Ensure time delta is never higher than inactivity period
- Whitelist reloading was limited to 100 entries

### Changed
- Run `go fmt`
- Use linear falloff when handling deadlocks
- Make some config values more sane
- Stop disabling http/1.1 keep-alive
- Reduce peer ID collision potential
- Structure of `transfer_ips` was reworked to include more data

### Removed
- Support for slots

## v0.3.0
### Added
- Add deserialization time logging
- Handle global freeleech
- Reactivate pruned torrents when a seeder announces

### Fixed
- Lock `mainConn` before closing it
- Only start HTTP server after initializing the database
- Fix database reload timeouts occurring during a slot cache verification
- Fix race condition causing users to appear inactive but leaving them in the peer hash

### Changed
- Force `Connection: close`
- Suppress superfluous HTTP panic logging
- Sleep before verifying slots for the first time
- Switch to a better used slots verification strategy
- Only load enabled users
- Flush the peer on every announce to update the last announce time

## v0.2.0
### Added
- Support for reverse proxy via `X-Real-Ip` header
- Deadlock handling
- Use multiple database connections

### Changed
- Run `go fmt`
- Flush torrent when a leecher becomes a seeder
- Log when done serializing
- Switch to `mymysql` (https://github.com/ziutek/mymysql)

## v0.1.0
### Added
- Initial release

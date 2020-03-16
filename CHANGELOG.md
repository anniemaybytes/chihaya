# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

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

### Changed
- Migrate to bencode library (https://github.com/zeebo/bencode)
- Explicitly ignore error handling on shutdown
- Code cleanup
- Remove obsolete TODOs
- Simplify ipAddr handling in announce
- Export ConfigMap from config
- Extract record into separate package
- Synchronously populate initial data from database into memory
- Explicitly Close() server itself on shutdown (https://github.com/golang/go/issues/10527)

## v0.6.0
### Added
- Base state for CHANGELOG

chihaya (kuroneko)
=======

Installation
-------------

chihaya requires Golang >= 1.14 and MariaDB >= 10.3.3.

```sh
go get
go build -v -o .bin/ ./cmd/...
```

Example systemd unit file:
```systemd
[Unit]
Description=chihaya
After=network.target mariadb.service

[Service]
WorkingDirectory=/opt/chihaya
ExecStart=/opt/chihaya/chihaya
RestartSec=30s
Restart=always
User=chihaya

[Install]
WantedBy=default.target
```

Alternatively, you can also build/use a docker container instead:

```sh
docker build . -t chihaya
docker run -d --restart=always --user 1001:1001 --network host --log-driver local -v ${PWD}:/app chihaya
```

Usage
-------------
Build process outputs several binary files. Each binary has its own flags, use 
`-h` or `--help` for detailed help on how to use them.

- `chihaya` - this is tracker itself
- `cc` - utility for manipulation of cache data
- `bencode` - utility for encoding and decoding between JSON and Bencode

Configuration
-------------

Configuration is done in `config.json`, which you'll need to create with the following format:

```json
{
  "database": {
    "username": "chihaya",
    "password": "",
    "database": "chihaya",
    "proto": "tcp",
    "addr": "127.0.0.1:3306",

    "deadlock_pause": 1,
    "deadlock_retries": 5
  },

  "channels": {
    "torrent": 5000,
    "user": 5000,
    "transfer_history": 5000,
    "transfer_ips": 5000,
    "snatch": 25
  },

  "intervals": {
    "announce": 1800,
    "min_announce": 900,
    "peer_inactivity": 3900,
    "announce_drift": 300,
    "scrape": 900,

    "database_reload": 45,
    "database_serialize": 68,
    "purge_inactive_peers": 120,

    "flush": 3
  },

  "addr": ":34000",

  "admin_token": "",
  "proxy": "",

  "record": false,
  "scrape": true,
  "log_flushes": true,
  "strict_port": false,
  "flush_groups": false,

  "numwant": 25,
  "max_numwant": 50
}
```

- `database`
    - `username` - username to use when connecting to database
    - `password` - password for user specified
    - `database` - database name
    - `proto` - protocol to use when connecting to database, can be `unix` or `tcp`
    - `addr` - address to find database at, either absolute path for `unix` or 
    `ip:port` for `tcp`
    - `deadlock_pause` - time in seconds to wait between retries on deadlock, ramps up 
    linearly with each attempt from this value
    - `deadlock_retries` - how many times should we retry on deadlock
- `channels` - channel holds raw data for injection to SQL statement on flush
    - `torrent` - maximum size of channel holding changes to `torrents` table
    - `user` - maximum size of channel holding changes to `users_main` table
    - `transfer_history` - maximum size of channel holding changes to `transfer_history`
    - `transfer_ips` - maximum size of channel holding changes to `transfer_ips`
    - `snatch`: maximum size of channels holding snatches for `transfer_history`
- `intervals` - all values are in seconds
    - `announce` - default announce `interval` given to clients
    - `min_announce` - minimum `min_interval` between announces that clients should respect
    - `peer_inactivity` - time after which peer is considered dead, recommended to be 
    `(min_announce * 2) + (announce_drift * 2)`
    - `announce_drift` - maximum announce drift to incorporate in default `interval` 
    sent to client
    - `scrape` - default scrape `interval` given to clients
    - `database_reload` - time between reloads of user and torrent data from database
    - `database_serialize` - time between database serializations to cache
    - `purge_inactive_peers` - time between peers older than `peer_inactivity` are flushed 
    from database and memory
    - `flush` - time between database flushes when channel is used in less than 50%
- `addr` - address to which we should listen for HTTP requests
- `admin_token` - administrative token used in `Authorization` header to access advanced 
prometheus statistics
- `proxy` - header name to look for user's real IP address, for example `X-Real-Ip`
- `record` - enables or disables JSON recorder of announces
- `scrape` - enables or disables `/scrape` endpoint which allows clients to get peers count 
without sending announce
- `log_flushes` - whether to log all database flushes performed
- `strict_port` - if enabled then announces where client advertises port outside range 
`1024-65535` will be failed
- `flush_groups` - whether to update groups timestamp whenever related torrent is flushed
- `numwant` - Default number of peers sent on announce if otherwise not explicitly specified 
by client
- `max_numwant` - Maximum number of peers that tracker will send per single announce, even if
client requests more

Recorder
-------------

If `record` is true, chihaya will save all successful announce events to a file under 
`events` directory. The files will have a format of `events_YYYY-MM-DDTHH.json` and are
split every hour for easier analysis. Every line in a file should be treated as a separate
JSON object. Below is a definition on how to read the data:

```text
[torrentId, userId, ipAddr, port, event, seeding, rawUp, rawDown, up, down, left] 
```

- `torrentId` - the ID of torrent being announced
- `userId` - the ID of user making the announce
- `ipAddr` - IP address of peer (this might _not_ be an address from which request was sent)
- `port` - port the peer listens on as resolved by tracker; this might be an invalid or
closed port, the tracker performs no validation on it
- `event` - the event as given by client; for regular announces it is empty
- `seeding` - whether tracker recognizes peer as seeder or leecher, either `1` or `0`
- `rawUp` - delta of uploaded between announces for this peer, in bytes
- `rawDown` - delta of downloaded between announces for this peer, in bytes
- `up` - number of `uploaded` bytes, as reported by client for this announce
- `down` - number of `downloaded` bytes, as reported by client for this announce
- `left` - number of `left` bytes, as reported by client for this announce; if this is more
than 0 then tracker sees this peer as leecher and `seeding` should be `0`

Flowcharts
-------------

#### IP resolve
```text
`ipv4` exists ->
    yes -> is it public?
        yes -> `ip` exists?
            yes -> is it the same as `ipv4`?
                yes -> use `ipv4`
                no  -> failure
            no -> use `ipv4`
        no  -> continue
    no  -> `ip` exists?
        yes -> is it public?
            yes -> use `ip`
            no  -> continue
        no  -> is proxy allowed?
            yes -> is configured header present?
                yes -> use header
                no  -> continue
            no  -> use socket
```

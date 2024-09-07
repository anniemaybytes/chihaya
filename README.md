chihaya (kuroneko)
=======

Installation
-------------

chihaya requires Go >= 1.21 and MariaDB >= 10.3.3

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

Chihaya is designed to be used behind reverse proxy (such as `nginx`) that can provide TLS termination as well as other
features such as rate limiting.

Usage of compression (such as `gzip`) is dicouraged as responses are usually quite small (especially when `compact` 
is requested), resulting in unnecessary overhead for zero gain.

Configuration
-------------

Configuration is done in `config.json`, which you'll need to create with the following schema:

```json
{
  "$id": "config.schema.json",
  "$schema": "https://json-schema.org/draft/2020-12/schema",

  "type": "object",
  "properties": {
    "database": {
      "type": "object",
      "properties": {
        "dsn": {
          "description": "Data Source Name at which to find database",
          "type": "string",
          "default": "chihaya:@tcp(127.0.0.1:3306)/chihaya"
        },
        "deadlock_pause": {
          "description": "Time in seconds to wait between retries on deadlock, ramps up linearly with each attempt from this value",
          "type": "integer",
          "default": 1
        },
        "deadlock_retries": {
          "description": "How many times should we retry on deadlock",
          "type": "integer",
          "default": 5
        }
      }
    },
    "channels": {
      "description": "Configures maximum size for various data channels",
      "type": "object",
      "properties": {
        "torrents": {
          "type": "integer",
          "default": 5000
        },
        "users": {
          "type": "integer",
          "default": 5000
        },
        "transfer_history": {
          "type": "integer",
          "default": 5000
        },
        "transfer_ips": {
          "type": "integer",
          "default": 5000
        },
        "snatches": {
          "type": "integer",
          "default": 25
        }
      }
    },
    "intervals": {
      "type": "object",
      "properties": {
        "announce": {
          "description": "Base value of interval given to clients in announce response (in seconds)",
          "type": "integer",
          "default": 1800
        },
        "min_announce": {
          "description": "Value of min_interval given to clients in announce response (in seconds)",
          "type": "integer",
          "default": 900
        },
        "announce_drift": {
          "description": "Maximum drift (in seconds) to be applied over base announce interval to help in spreading load",
          "type": "integer",
          "default": 300
        },
        "peer_inactivity": {
          "description": "Maximum time (in seconds) after which peer will be considered inactive; should be at least double the interval (incl. drift)",
          "type": "integer",
          "default": 4200
        },
        "scrape": {
          "description": "Value of min_request_interval given to clients in scrape response (in seconds); not all clients respect it",
          "type": "integer",
          "default": 900
        },
        "database_reload": {
          "description": "Time (in seconds) between fresh user and torrent data is reloaded from database",
          "type": "integer",
          "default": 45
        },
        "database_serialize": {
          "description": "Time (in seconds) between serializations of in-memory peer data to cache file",
          "type": "integer",
          "default": 68
        },
        "purge_inactive_peers": {
          "description": "Time (in seconds) between thread is executed to scan and purge inactive peers from memory and database",
          "type": "integer",
          "default": 120
        },
        "flush": {
          "description": "Time (in seconds) to delay next flush if data channel was consumed in less than 50% on previous flush",
          "type": "integer",
          "default": 3
        }
      }
    },
    "http": {
      "type": "object",
      "properties": {
        "addr": {
          "description": "Address on which FastHTTP server will listen for requests",
          "type": "string",
          "default": ":34000"
        },
        "proxy_header": {
          "description": "Name of header which will be used to replace IP address of connection when running behind reverse proxy",
          "type": "string",
          "default": ""
        },
        "timeout": {
          "description": "Configures timeout values for FastHTTP",
          "type": "object",
          "properties": {
            "read": {
              "description": "Time (in milliseconds) to fully read request content from socket",
              "type": "integer",
              "default": 300
            },
            "write": {
              "description": "Time (in milliseconds) to perform single write operation on socket",
              "type": "integer",
              "default": 300
            },
            "idle": {
              "description": "Time (in seconds) to keep connection open for Keep-Alive requests",
              "type": "integer",
              "default": 300
            }
          }
        }
      }
    },
    "announce": {
      "type": "object",
      "properties": {
        "strict_port": {
          "description": "Whether to reject announces when client reports it is listening for peer connections on ports below 1024",
          "type": "boolean",
          "default": false
        },
        "numwant": {
          "description": "Number of peers given to client in announce response, unless client explicitly requests other value",
          "type": "integer",
          "default": 25
        },
        "max_numwant": {
          "description": "Maximum number of peers tracker will ever give in single announce response, even if client asks for more",
          "type": "integer",
          "default": 50
        }
      }
    },
    "record_announces": {
      "description": "Whether to enable recording of successful announces (for debugging or analysis purposes); might negatively impact performance",
      "type": "boolean",
      "default": false
    },
    "enable_scrape": {
      "description": "Whether to enable BEP-48 extension",
      "type": "boolean",
      "default": true
    },
    "enable_metrics": {
      "description": "Whether to enable Prometheus metrics endpoint",
      "type": "boolean",
      "default": false
    },
    "log_flushes": {
      "description": "Whether to log details about database flushes to standard output",
      "type": "boolean",
      "default": true
    }
  }
}
```

Recorder
-------------

Chihaya supports saving all successful announce events to a file under 
`events` directory. The files will have a format of `events_YYYY-MM-DDTHH.csv` and are
split hourly for easier analysis.

Database scheme
-------------
Supported database scheme can be located in `database/schema.sql`.

Example data from fixtures can be consulted for additional help.

Flowcharts
-------------

#### IP resolution
![IP resolution flowchart](.gitea/images/flowcharts/ip.png)

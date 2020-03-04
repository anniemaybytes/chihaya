chihaya (kuroneko)
=======

Installation
-------------

chihaya requires Golang >= 1.14 and MariaDB >= 10.3.3.

```
$ go get
$ go build
```

Additionally, you may pass tags during build to control which functions you want to enable. Supported tags are:
- scrape: Enables optional support for /scrape endpoint
- record: Enables simple experimental JSON recorder of announce events to flat file

Example:
```
$ go build -tags "scrape record"
```

Configuration
-------------

Timing configuration is currently hardcoded in `config/config.go`. Edit that and recompile.

Database configuration is done in `config.json`, which you'll need to create with the following format:

```json
{
	"database": {
		"username": "user",
		"password": "pass",
		"database": "database",
		"proto": "unix",
		"addr": "/var/run/mysqld/mysqld.sock",
		"encoding": "utf8"
	},

	"addr": ":34000",
	"admin_token": "admintoken",
	"proxy": null
}
```

`addr` specifies the address to bind the server to. Possible values for `database.proto` are `unix` and `tcp`. If protocol is `tcp` then `addr` should be in form of `ip:port`
`admin_token` is for advanced metrics in /metrics endpoint. Use with `/metrics?token=<your token>`.
`proxy` decides which proxy headers to check for IP, if a valid IP cannot be found in parameters. Can be `null` or a valid header.

Running
-------

Simply execute it via `./chihaya` to run normally.

Additional flags:
- `profile` - generate profiling data for pprof into chihaya.cpu

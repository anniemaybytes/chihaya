chihaya (kuroneko)
=======

Installation
-------------

chihaya requires Golang >= 1.14 and MariaDB >= 10.3.3.

```
$ go get
$ go build
```

Configuration
-------------

Timing configuration is currently hardcoded in `config/config.go`. Edit that and recompile.

Database configuration is done in `config.json`, which you'll need to create with the following format:

```json
{
	"database": {
		"username": "chihaya",
		"password": "",
		"database": "chihaya",
		"proto": "tcp",
		"addr": "127.0.0.1:3306"
	},

	"addr": ":34000",

	"admin_token": "",
	"proxy": "",

	"record": false,
	"scrape": true,
	"log_flushes": true,
	"strict_port": false
}
```

- `addr` specifies the address to bind the server to. Possible values for `database.proto` are `unix` and `tcp`. If protocol is `tcp` then `addr` should be in form of `ip:port`
- `admin_token` is for advanced metrics in `/metrics` endpoint. Can be empty string `""` to disable.
- `proxy` decides which proxy headers to check for IP, if a valid IP cannot be found in parameters. Can be empty string `""` to disable or a valid header name to enable.
- `scrape` enables optional support for /scrape endpoint. Optional, defaults to `false`.
- `record` enables simple experimental JSON recorder of announce events to flat file. Optional, defaults to `true`.
- `log_flushes` enables logging of all flush actions, defaults to `true`
- `strict_port` enables strict port checking for announces (1024-65535), defaults to `false`

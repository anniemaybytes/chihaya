chihaya (kuroneko)
=======

Installation
-------------

chihaya requires Golang. Version 1.11+ is recommended.

```
go get
go build
```

Additionally, you may pass tags during build to control which functions you want to enable. Supported tags are:
- scrape: Enables optional support for /scrape endpoint
- record: Enables simple experimental JSON recorded of announce events to flat file

Example:
```
go build -tags "scrape record"
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

	"addr": ":34000"
}
```

`addr` specifies the address to bind the server to. Possible values for `database.proto` are `unix` and `tcp`.

Running
-------

`./chihaya` to run normally, `./chihaya -profile` to generate pprof data for analysis.

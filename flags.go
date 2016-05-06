// flags
package main

import (
	"github.com/codegangsta/cli"
)

var serviceFlags = []cli.Flag{
	cli.StringFlag{
		Name:        "id, i",
		Usage:       "unique node id, a random hash will be generated if not specified",
		Destination: &id,
	},
	cli.StringFlag{
		Name:        "slaveof, m",
		Usage:       "master node url, format: host:port. master if empty",
		Destination: &slaveOf,
	},
	cli.IntFlag{
		Name:        "port_http, P",
		Value:       1103,
		Usage:       "http port",
		Destination: &portHttp,
	},
	cli.BoolFlag{
		Name:        "enable_https, e",
		Usage:       "true to enable https, false by default",
		Destination: &enableHttps,
	},
	cli.IntFlag{
		Name:        "port_https, p",
		Value:       2015,
		Usage:       "https port",
		Destination: &portHttps,
	},
	cli.StringFlag{
		Name:        "host_https, l",
		Value:       "127.0.0.1",
		Usage:       "https host name. [::] for all",
		Destination: &hostHttps,
	},
	cli.StringFlag{
		Name:        "cert_file, c",
		Usage:       "cert file path, search path: ~/.netdata/cert.crt, /etc/netdata/cert.crt",
		Destination: &certFile,
	},
	cli.StringFlag{
		Name:        "key_file, k",
		Usage:       "key file path, search path: ~/.netdata/key.key, /etc/netdata/key.key",
		Destination: &keyFile,
	},
	cli.StringFlag{
		Name:        "conf_file, C",
		Usage:       "configuration file path, search path: ~/.netdata/netdata.json, /etc/netdata/netdata.json",
		Destination: &confFile,
	},
	cli.StringFlag{
		Name:        "data_file, d",
		Usage:       "master data file path, ignored by slave nodes, search path: ~/.netdata/netdata_master.json, /etc/netdata/netdata_master.json",
		Destination: &dataFile,
	},
}

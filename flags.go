// flags
package main

import (
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/elgs/gojq"
	"github.com/satori/go.uuid"
)

type CliNetDataService struct {
	Id          string
	SlaveOf     string
	EnableHttp  bool // true
	PortHttp    int
	HostHttp    string // "127.0.0.1"
	EnableHttps bool
	PortHttps   int
	HostHttps   string
	CertFile    string
	KeyFile     string
	ConfFile    string
	DataFile    string
}

func (this *CliNetDataService) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "id, i",
			Usage:       "unique node id, a random hash will be generated if not specified",
			Destination: &this.Id,
		},
		cli.StringFlag{
			Name:        "slaveof, m",
			Usage:       "master node url, format: host:port. master if empty",
			Destination: &this.SlaveOf,
		},
		cli.IntFlag{
			Name:        "port_http, P",
			Value:       1103,
			Usage:       "http port",
			Destination: &this.PortHttp,
		},
		cli.BoolFlag{
			Name:        "enable_https, e",
			Usage:       "true to enable https, false by default",
			Destination: &this.EnableHttps,
		},
		cli.IntFlag{
			Name:        "port_https, p",
			Value:       2015,
			Usage:       "https port",
			Destination: &this.PortHttps,
		},
		cli.StringFlag{
			Name:        "host_https, l",
			Value:       "127.0.0.1",
			Usage:       "https host name. [::] for all",
			Destination: &this.HostHttps,
		},
		cli.StringFlag{
			Name:        "cert_file, c",
			Usage:       "cert file path, search path: ~/.netdata/cert.crt, /etc/netdata/cert.crt",
			Destination: &this.CertFile,
		},
		cli.StringFlag{
			Name:        "key_file, k",
			Usage:       "key file path, search path: ~/.netdata/key.key, /etc/netdata/key.key",
			Destination: &this.KeyFile,
		},
		cli.StringFlag{
			Name:        "conf_file, C",
			Usage:       "configuration file path, search path: ~/.netdata/netdata.json, /etc/netdata/netdata.json",
			Destination: &this.ConfFile,
		},
		cli.StringFlag{
			Name:        "data_file, d",
			Usage:       "master data file path, ignored by slave nodes, search path: ~/.netdata/netdata_master.json, /etc/netdata/netdata_master.json",
			Destination: &this.DataFile,
		},
	}
}

func (this *CliNetDataService) LoadConfigs(c *cli.Context) {
	// read config file
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	this.LoadConfig("/etc/netdata/netdata.json", c)
	this.LoadConfig(usr.HomeDir+"/.netdata/netdata.json", c)
	this.LoadConfig(pwd+"/netdata.json", c)
	this.LoadConfig(this.ConfFile, c)
	if strings.TrimSpace(this.Id) == "" {
		this.Id = strings.Replace(uuid.NewV4().String(), "-", "", -1)
	}
}

func (this *CliNetDataService) LoadConfig(file string, c *cli.Context) {
	jqConf, err := gojq.NewFileQuery(file)
	if err != nil {
		//ignore
		return
	}
	if !c.IsSet("id") {
		v, err := jqConf.QueryToString("id")
		if err == nil {
			this.Id = v
		}
	}
	if !c.IsSet("slaveof") {
		v, err := jqConf.QueryToString("slaveof")
		if err == nil {
			this.SlaveOf = v
		}
	}
	if !c.IsSet("port_http") {
		v, err := jqConf.QueryToInt64("port_http")
		if err == nil {
			this.PortHttp = int(v)
		}
	}
	if !c.IsSet("enable_http") {
		v, err := jqConf.QueryToBool("enable_http")
		if err == nil {
			this.EnableHttp = v
		}
	}
	if !c.IsSet("enable_https") {
		v, err := jqConf.QueryToBool("enable_https")
		if err == nil {
			this.EnableHttps = v
		}
	}
	if !c.IsSet("port_https") {
		v, err := jqConf.QueryToInt64("port_https")
		if err == nil {
			this.PortHttps = int(v)
		}
	}
	if !c.IsSet("host_https") {
		v, err := jqConf.QueryToString("host_https")
		if err == nil {
			this.HostHttps = v
		}
	}
	if !c.IsSet("cert_file") {
		v, err := jqConf.QueryToString("cert_file")
		if err == nil {
			this.CertFile = v
		}
	}
	if !c.IsSet("key_file") {
		v, err := jqConf.QueryToString("key_file")
		if err == nil {
			this.KeyFile = v
		}
	}
	if !c.IsSet("conf_file") {
		v, err := jqConf.QueryToString("conf_file")
		if err == nil {
			this.ConfFile = v
		}
	}
	if !c.IsSet("data_file") {
		v, err := jqConf.QueryToString("data_file")
		if err == nil {
			this.DataFile = v
		}
	}
}

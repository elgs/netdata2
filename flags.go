// flags
package main

import (
	"strings"

	"github.com/elgs/gojq"
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
)

type CliService struct {
	Id           string
	Master       string
	EnableHttp   bool // true
	HttpPort     int
	HttpHost     string // "127.0.0.1"
	EnableHttps  bool
	HttpsPort    int
	HttpsHost    string
	CertFile     string
	KeyFile      string
	ConfFile     string
	DataFile     string
	Secret       string
	MailHost     string
	MailPort     int
	MailUsername string
	MailPassword string
}

func (this *CliService) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "id, i",
			Usage:       "unique node id, a random hash will be generated if not specified",
			Destination: &this.Id,
		},
		cli.StringFlag{
			Name:        "master, m",
			Usage:       "master node url, format: host:port. master if empty",
			Destination: &this.Master,
		},
		cli.IntFlag{
			Name:        "http_port, P",
			Value:       1103,
			Usage:       "http port",
			Destination: &this.HttpPort,
		},
		cli.BoolFlag{
			Name:        "enable_https, e",
			Usage:       "true to enable https, false by default",
			Destination: &this.EnableHttps,
		},
		cli.IntFlag{
			Name:        "https_port, p",
			Value:       2015,
			Usage:       "https port",
			Destination: &this.HttpsPort,
		},
		cli.StringFlag{
			Name:        "https_host, l",
			Value:       "127.0.0.1",
			Usage:       "https host name. [::] for all",
			Destination: &this.HttpsHost,
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
			Value:       homeDir + "/.netdata/netdata_master.json",
			Usage:       "master data file path, ignored by slave nodes, search path: ~/.netdata/netdata_master.json",
			Destination: &this.DataFile,
		},
		cli.StringFlag{
			Name:        "secret, z",
			Usage:       "secret password for server client communication.",
			Destination: &this.Secret,
		},
	}
}

func (this *CliService) LoadConfigs(c *cli.Context) {
	this.LoadConfig("/etc/netdata/netdata.json", c)
	this.LoadConfig(homeDir+"/.netdata/netdata.json", c)
	this.LoadConfig(pwd+"/netdata.json", c)
	this.LoadConfig(this.ConfFile, c)
	if strings.TrimSpace(this.Id) == "" {
		this.Id = strings.Replace(uuid.NewV4().String(), "-", "", -1)
	}
}

func (this *CliService) LoadSecrets(c *cli.Context) {
	this.LoadSecret("/etc/netdata/netdata.json", c)
	this.LoadSecret(homeDir+"/.netdata/netdata.json", c)
	this.LoadSecret(pwd+"/netdata.json", c)
	this.LoadSecret(this.ConfFile, c)
}

func (this *CliService) LoadConfig(file string, c *cli.Context) {
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
	if !c.IsSet("master") {
		v, err := jqConf.QueryToString("master")
		if err == nil {
			this.Master = v
		}
	}
	if !c.IsSet("http_port") {
		v, err := jqConf.QueryToInt64("http_port")
		if err == nil {
			this.HttpPort = int(v)
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
	if !c.IsSet("https_port") {
		v, err := jqConf.QueryToInt64("https_port")
		if err == nil {
			this.HttpsPort = int(v)
		}
	}
	if !c.IsSet("https_host") {
		v, err := jqConf.QueryToString("https_host")
		if err == nil {
			this.HttpsHost = v
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
	if !c.IsSet("secret") {
		v, err := jqConf.QueryToString("secret")
		if err == nil {
			this.Secret = v
		}
	}
	if !c.IsSet("mail_port") {
		v, err := jqConf.QueryToInt64("mail_port")
		if err == nil {
			this.MailPort = int(v)
		}
	}
	if !c.IsSet("mail_host") {
		v, err := jqConf.QueryToString("mail_host")
		if err == nil {
			this.MailHost = v
		}
	}
	if !c.IsSet("mail_username") {
		v, err := jqConf.QueryToString("mail_username")
		if err == nil {
			this.MailUsername = v
		}
	}
	if !c.IsSet("mail_password") {
		v, err := jqConf.QueryToString("mail_password")
		if err == nil {
			this.MailPassword = v
		}
	}
}

func (this *CliService) LoadSecret(file string, c *cli.Context) {
	jqConf, err := gojq.NewFileQuery(file)
	if err != nil {
		//ignore
		return
	}
	if !c.IsSet("secret") {
		v, err := jqConf.QueryToString("secret")
		if err == nil {
			this.Secret = v
		}
	}
}

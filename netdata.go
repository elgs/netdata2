package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/elgs/gojq"
	"github.com/elgs/gorest2"
)

var slaveOf string
var enableHttp bool = true
var portHttp int
var hostHttp string = "127.0.0.1"
var enableHttps bool
var portHttps int
var hostHttps string
var certFile string
var keyFile string
var confFile string
var dataFile string
var shutdownPort int

func loadConfigs(c *cli.Context) {
	// read config file
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	loadConfig("/etc/netdata/netdata.json", c)
	loadConfig(usr.HomeDir+"/.netdata/netdata.json", c)
	loadConfig(confFile, c)
}

func loadConfig(file string, c *cli.Context) {
	jqConf, err := gojq.NewFileQuery(file)
	if err != nil {
		//ignore
		return
	}
	if !c.IsSet("slaveof") {
		v, err := jqConf.QueryToString("slaveof")
		if err == nil {
			slaveOf = v
		}
	}
	if !c.IsSet("enable_http") {
		v, err := jqConf.QueryToBool("enable_http")
		if err == nil {
			enableHttp = v
		}
	}
}

func loadMasterData(file string) {}

func main() {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		// cleanup code here
		done <- true
	}()

	app := cli.NewApp()
	app.Name = "netdata"
	app.Usage = "An SQL backend for the web."
	app.Version = "0.0.1"
	app.Action = func(c *cli.Context) {
		loadConfigs(c)
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "slaveof, m",
			Usage:       "master node url, format: https://host:port. master if empty",
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
		cli.IntFlag{
			Name:        "shutdown_port, x",
			Value:       2014,
			Usage:       "port listening shutdown command",
			Destination: &shutdownPort,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "service",
			Aliases: []string{"s"},
			Usage:   "service commands",
			Subcommands: []cli.Command{
				{
					Name:  "start",
					Usage: "start serviec",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						if len(strings.TrimSpace(slaveOf)) > 0 {
							// load data from master if slave
							fmt.Println("slaveOf:", slaveOf)
						} else {
							// load data from data file if master
							fmt.Println("master")
						}
						// serve
						gorest2.RegisterHandler("/sys/shutdown", func(w http.ResponseWriter, r *http.Request) {
							if strings.HasPrefix(r.RemoteAddr, "127.0.0.1:") {
								done <- true
							} else {
								fmt.Fprintln(w, "Attack!!!")
							}
						})
						serve()
						<-done
					},
				},
				{
					Name:  "stop",
					Usage: "stop serviec",
					Action: func(c *cli.Context) {
						if len(c.Args()) > 0 {
							_, err := http.Post(fmt.Sprint("http://127.0.0.1:", c.Args()[0], "/sys/shutdown"), "text/plain", nil)
							if err != nil {
								fmt.Println(err)
							}
						} else {
							fmt.Println("Usage:netdata service stop <shutdown_port>")
						}
					},
				},
			},
		},
		{
			Name:    "datanode",
			Aliases: []string{"dn"},
			Usage:   "data node commands",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list all data nodes",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println(slaveOf)
					},
				},
				{
					Name:  "add",
					Usage: "add a new data node",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "update",
					Usage: "add an existing data node",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing data node",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("removed task template: ", c.Args().First())
					},
				},
			},
		},
		{
			Name:    "apinode",
			Aliases: []string{"an"},
			Usage:   "api node commands",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list all api nodes",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "add",
					Usage: "add a new api node",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "update",
					Usage: "add an existing api node",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing api node",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("removed task template: ", c.Args().First())
					},
				},
			},
		},
		{
			Name:    "app",
			Aliases: []string{"a"},
			Usage:   "app commands",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list all apps",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "add",
					Usage: "add a new app",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "update",
					Usage: "add an existing app",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing app",
					Action: func(c *cli.Context) {
						loadConfigs(c)
						println("removed task template: ", c.Args().First())
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

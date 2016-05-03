package main

import (
	"encoding/json"
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
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

type WsCommand struct {
	Type string
	Data string
	Meta map[string]interface{}
}

var id string
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
var wsConns = make(map[string]*websocket.Conn)

func loadConfigs(c *cli.Context) {
	// read config file
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	loadConfig("/etc/netdata/netdata.json", c)
	loadConfig(usr.HomeDir+"/.netdata/netdata.json", c)
	loadConfig(confFile, c)
	if strings.TrimSpace(id) == "" {
		id = strings.Replace(uuid.NewV4().String(), "-", "", -1)
	}
}

func loadConfig(file string, c *cli.Context) {
	jqConf, err := gojq.NewFileQuery(file)
	if err != nil {
		//ignore
		return
	}
	if !c.IsSet("id") {
		v, err := jqConf.QueryToString("id")
		if err == nil {
			id = v
		}
	}
	if !c.IsSet("slaveof") {
		v, err := jqConf.QueryToString("slaveof")
		if err == nil {
			slaveOf = v
		}
	}
	if !c.IsSet("port_http") {
		v, err := jqConf.QueryToInt64("port_http")
		if err == nil {
			portHttp = int(v)
		}
	}
	if !c.IsSet("enable_http") {
		v, err := jqConf.QueryToBool("enable_http")
		if err == nil {
			enableHttp = v
		}
	}
	if !c.IsSet("enable_https") {
		v, err := jqConf.QueryToBool("enable_https")
		if err == nil {
			enableHttps = v
		}
	}
	if !c.IsSet("port_https") {
		v, err := jqConf.QueryToInt64("port_https")
		if err == nil {
			portHttps = int(v)
		}
	}
	if !c.IsSet("host_https") {
		v, err := jqConf.QueryToString("host_https")
		if err == nil {
			hostHttps = v
		}
	}
	if !c.IsSet("cert_file") {
		v, err := jqConf.QueryToString("cert_file")
		if err == nil {
			certFile = v
		}
	}
	if !c.IsSet("key_file") {
		v, err := jqConf.QueryToString("key_file")
		if err == nil {
			keyFile = v
		}
	}
	if !c.IsSet("conf_file") {
		v, err := jqConf.QueryToString("conf_file")
		if err == nil {
			confFile = v
		}
	}
	if !c.IsSet("data_file") {
		v, err := jqConf.QueryToString("data_file")
		if err == nil {
			dataFile = v
		}
	}
}

func loadMasterData(file string) {}

func main() {

	sigs := make(chan os.Signal, 1)
	wsDrop := make(chan bool, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case sig := <-sigs:
				fmt.Println()
				fmt.Println(sig)
				// cleanup code here
				done <- true
			case <-wsDrop:
				fmt.Println("ws dropped.")
			}
		}
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

	app.Commands = []cli.Command{
		{
			Name:    "service",
			Aliases: []string{"s"},
			Usage:   "service commands",
			Subcommands: []cli.Command{
				{
					Name:  "start",
					Usage: "start serviec",
					Flags: app.Flags,
					Action: func(c *cli.Context) {
						loadConfigs(c)
						if len(strings.TrimSpace(slaveOf)) > 0 {
							// load data from master if slave
							c, _, err := websocket.DefaultDialer.Dial("wss://"+slaveOf+"/sys/ws", nil)
							if err != nil {
								fmt.Println(err)
								wsDrop <- true
							}
							go func() {
								defer c.Close()
								defer func() { wsDrop <- true }()
								for {
									_, message, err := c.ReadMessage()
									if err != nil {
										log.Println("read:", err)
										return
									}
									log.Printf("recv: %s", message)
								}
							}()

							regCommand := WsCommand{
								Type: "Register",
								Data: id,
							}

							// Register
							if err := c.WriteJSON(regCommand); err != nil {
								fmt.Println(err)
								wsDrop <- true
							}
						} else {
							// load data from data file if master
							gorest2.RegisterHandler("/sys/ws", func(w http.ResponseWriter, r *http.Request) {
								conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
								if err != nil {
									http.Error(w, err.Error(), http.StatusInternalServerError)
									return
								}

								go func(c *websocket.Conn) {
									defer conn.Close()
									for {
										_, message, err := c.ReadMessage()
										if err != nil {
											fmt.Println(err)
											c.Close()
											for k, v := range wsConns {
												if v == conn {
													delete(wsConns, k)
													break
												}
											}
											break
										}
										wsCommand := &WsCommand{}
										json.Unmarshal(message, wsCommand)
										if wsCommand.Type == "Register" {
											wsConns[wsCommand.Data] = conn
										}
										fmt.Println(wsCommand)
									}
								}(conn)
							})
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

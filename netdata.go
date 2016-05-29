package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/elgs/gorest2"
	"github.com/gorilla/websocket"
)

var slaveConn *websocket.Conn
var wsConns = make(map[string]*websocket.Conn)
var masterData MasterData
var pwd string
var homeDir string

func loadMasterData(file string)             {}
func storeMasterData(masterData *MasterData) {}

var service = &CliService{
	EnableHttp: true,
	HostHttp:   "127.0.0.1",
}

func main() {
	// read config file
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	homeDir = usr.HomeDir
	pwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
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
				RegisterToMaster(wsDrop)
			}
		}
	}()

	app := cli.NewApp()
	app.Name = "netdata"
	app.Usage = "An SQL backend for the web."
	app.Version = "0.0.1"
	app.Action = func(c *cli.Context) error {
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:    "service",
			Aliases: []string{"s"},
			Usage:   "service commands",
			Subcommands: []cli.Command{
				{
					Name:  "start",
					Usage: "start service",
					Flags: service.Flags(),
					Action: func(c *cli.Context) error {
						service.LoadConfigs(c)
						if len(strings.TrimSpace(service.Master)) > 0 {
							// load data from master if slave
							RegisterToMaster(wsDrop)
						} else {
							// load data from data file if master
							gorest2.RegisterHandler("/sys/ws", func(w http.ResponseWriter, r *http.Request) {
								conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
								if err != nil {
									http.Error(w, err.Error(), http.StatusInternalServerError)
									return
								}

								go func(c *websocket.Conn) {
									defer c.Close()
									for {
										_, message, err := c.ReadMessage()
										if err != nil {
											masterData.RemoveApiNode(c.RemoteAddr().String())
											c.Close()
											log.Println(c.RemoteAddr(), "dropped.")
											for k, v := range wsConns {
												if v == c {
													delete(wsConns, k)
													break
												}
											}
											break
										}
										// Master to process command from client web socket channels.
										err = processWsCommandMaster(c, message)
										if err != nil {
											log.Println(err)
										}
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
						// cli
						gorest2.RegisterHandler("/sys/cli", func(w http.ResponseWriter, r *http.Request) {
							res, err := ioutil.ReadAll(r.Body)
							if err != nil {
								fmt.Fprint(w, err.Error())
								return
							}
							if service.Master == "" {
								// Master to process commands from cli interface.
								result, err := processCliCommand(res)
								if err != nil {
									fmt.Fprint(w, err.Error())
									return
								}
								fmt.Fprint(w, result)
							} else {
								cliCommand := &Command{}
								json.Unmarshal(res, cliCommand)
								// Slave to forward cli command to master.
								response, err := sendCliCommand(service.Master, cliCommand)
								if err != nil {
									fmt.Fprint(w, err.Error())
									return
								}
								output := string(response)
								fmt.Fprint(w, output)
							}
						})
						serve(service)
						<-done
						return nil
					},
				},
				{
					Name:  "stop",
					Usage: "stop service",
					Action: func(c *cli.Context) error {
						if len(c.Args()) > 0 {
							_, err := http.Post(fmt.Sprint("http://127.0.0.1:", c.Args()[0], "/sys/shutdown"), "text/plain", nil)
							if err != nil {
								fmt.Println(err)
								return err
							}
						} else {
							fmt.Println("Usage: netdata service stop <shutdown_port>")
						}
						return nil
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
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.BoolTFlag{
							Name:  "full, f",
							Usage: "show a full list of data nodes",
						},
						cli.BoolTFlag{
							Name:  "compact, c",
							Usage: "show a compact list of data nodes",
						}},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						full := c.IsSet("full")
						compact := c.IsSet("compact")
						mode := "normal"
						if compact {
							mode = "compact"
						} else if full {
							mode = "full"
						}
						cliDnListCommand := &Command{
							Type: "CLI_DN_LIST",
							Data: mode,
						}
						response, err := sendCliCommand(master, cliDnListCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
				{
					Name:  "add",
					Usage: "add a new data node",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the data node",
						},
						cli.StringFlag{
							Name:  "host, H",
							Usage: "hostname of the data node",
						},
						cli.IntFlag{
							Name:  "port, P",
							Value: 3306,
							Usage: "port number of the data node",
						},
						cli.StringFlag{
							Name:  "user, u",
							Usage: "username of the data node",
						},
						cli.StringFlag{
							Name:  "pass, p",
							Usage: "password of the node",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the data node",
						},
					},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						dataNode := &DataNode{
							Name:     c.String("name"),
							Host:     c.String("host"),
							Port:     c.Int("port"),
							Username: c.String("user"),
							Password: c.String("pass"),
							Note:     c.String("note"),
						}
						dataNodeJSONBytes, err := json.Marshal(dataNode)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliDnAddCommand := &Command{
							Type: "CLI_DN_ADD",
							Data: string(dataNodeJSONBytes),
						}
						response, err := sendCliCommand(master, cliDnAddCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
				{
					Name:  "update",
					Usage: "update an existing data node",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the data node",
						},
						cli.StringFlag{
							Name:  "host, H",
							Usage: "hostname of the data node",
						},
						cli.IntFlag{
							Name:  "port, P",
							Value: 3306,
							Usage: "port number of the data node",
						},
						cli.StringFlag{
							Name:  "user, u",
							Usage: "username of the data node",
						},
						cli.StringFlag{
							Name:  "pass, p",
							Usage: "password of the node",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the data node",
						},
					},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						dataNode := &DataNode{
							Name:     c.String("name"),
							Host:     c.String("host"),
							Port:     c.Int("port"),
							Username: c.String("user"),
							Password: c.String("pass"),
							Note:     c.String("note"),
						}
						dataNodeJSONBytes, err := json.Marshal(dataNode)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliDnUpdateCommand := &Command{
							Type: "CLI_DN_UPDATE",
							Data: string(dataNodeJSONBytes),
						}
						response, err := sendCliCommand(master, cliDnUpdateCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing data node",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the data node",
						},
					},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						name := c.String("name")
						cliDnRemoveCommand := &Command{
							Type: "CLI_DN_REMOVE",
							Data: name,
						}
						response, err := sendCliCommand(master, cliDnRemoveCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
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
					Action: func(c *cli.Context) error {
						return nil
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
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.BoolTFlag{
							Name:  "full, f",
							Usage: "show a full list of apps",
						},
						cli.BoolTFlag{
							Name:  "compact, c",
							Usage: "show a compact list of apps",
						}},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						full := c.IsSet("full")
						compact := c.IsSet("compact")
						mode := "normal"
						if compact {
							mode = "compact"
						} else if full {
							mode = "full"
						}
						cliAppListCommand := &Command{
							Type: "CLI_APP_LIST",
							Data: mode,
						}
						response, err := sendCliCommand(master, cliAppListCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
				{
					Name:  "add",
					Usage: "add a new app",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the app",
						},
						cli.StringFlag{
							Name:  "datanode, d",
							Usage: "data node name",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the app",
						},
					},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						app := &App{
							Name:         c.String("name"),
							DataNodeName: c.String("datanode"),
							Note:         c.String("note"),
						}
						appJSONBytes, err := json.Marshal(app)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliAppAddCommand := &Command{
							Type: "CLI_APP_ADD",
							Data: string(appJSONBytes),
						}
						response, err := sendCliCommand(master, cliAppAddCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
				{
					Name:  "update",
					Usage: "update an existing app",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the app",
						},
						cli.StringFlag{
							Name:  "datanode, d",
							Usage: "data node name",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the app",
						},
					},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						app := &App{
							Name:         c.String("name"),
							DataNodeName: c.String("datanode"),
							Note:         c.String("note"),
						}
						appJSONBytes, err := json.Marshal(app)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliAppUpdateCommand := &Command{
							Type: "CLI_APP_UPDATE",
							Data: string(appJSONBytes),
						}
						response, err := sendCliCommand(master, cliAppUpdateCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing app",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the app",
						},
					},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						name := c.String("name")
						cliAppRemoveCommand := &Command{
							Type: "CLI_APP_REMOVE",
							Data: name,
						}
						response, err := sendCliCommand(master, cliAppRemoveCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
			},
		},
		{
			Name:  "show",
			Usage: "show commands",
			Subcommands: []cli.Command{
				{
					Name:  "master",
					Usage: "show master data",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "master, m",
							Value: "127.0.0.1:2015",
							Usage: "master node url, format: host:port. 127.0.0.1:2015 if empty",
						},
					},
					Action: func(c *cli.Context) error {
						master := c.String("master")
						cliShowMasterCommand := &Command{
							Type: "CLI_SHOW_MASTER",
						}
						response, err := sendCliCommand(master, cliShowMasterCommand)
						if err != nil {
							fmt.Println(err)
							return err
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
						return nil
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

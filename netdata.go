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
	"github.com/elgs/gostrgen"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

var slaveConn *websocket.Conn

var wsConns = make(map[string]*websocket.Conn)
var masterData MasterData
var apiNodes []ApiNode
var pwd string
var homeDir string

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
						if _, err := os.Stat(service.DataFile); os.IsNotExist(err) {
							fmt.Println(err)
						} else {
							masterDataBytes, err := ioutil.ReadFile(service.DataFile)
							if err != nil {
								return err
							}
							err = json.Unmarshal(masterDataBytes, &masterData)
							if err != nil {
								return err
							}
						}

						gorest2.GetDbo = MakeGetDbo("mysql", &masterData)

						if len(strings.TrimSpace(service.Master)) > 0 {
							// load data from master if slave
							RegisterToMaster(wsDrop)
						} else {
							// load data from data file if master
							StartJobs()
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
											err = RemoveApiNode(c.RemoteAddr().String())
											if err != nil {
												log.Println(err)
											}
											err = c.Close()
											if err != nil {
												log.Println(err)
											}
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
						// shutdown
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

						gorest2.RegisterHandler("/api", gorest2.RestFunc)

						// serve
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
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
						node := c.String("node")
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
						response, err := sendCliCommand(node, cliDnListCommand)
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
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
						node := c.String("node")
						id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
						dataNode := &DataNode{
							Id:       id,
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
						response, err := sendCliCommand(node, cliDnAddCommand)
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the data node",
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
						node := c.String("node")
						dataNode := &DataNode{
							Id:       c.String("id"),
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
						response, err := sendCliCommand(node, cliDnUpdateCommand)
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the data node",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := c.String("id")
						cliDnRemoveCommand := &Command{
							Type: "CLI_DN_REMOVE",
							Data: id,
						}
						response, err := sendCliCommand(node, cliDnRemoveCommand)
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
			Name:    "app",
			Aliases: []string{"a"},
			Usage:   "app commands",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list all apps",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
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
						node := c.String("node")
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
						response, err := sendCliCommand(node, cliAppListCommand)
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the app",
						},
						cli.StringFlag{
							Name:  "datanode, d",
							Usage: "data node id",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the app",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
						dbName, err := gostrgen.RandGen(16, gostrgen.LowerDigit, "", "")
						if err != nil {
							return err
						}
						app := &App{
							Id:         id,
							Name:       c.String("name"),
							DataNodeId: c.String("datanode"),
							DbName:     dbName,
							Note:       c.String("note"),
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
						response, err := sendCliCommand(node, cliAppAddCommand)
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the app",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the app",
						},
						cli.StringFlag{
							Name:  "datanode, d",
							Usage: "data node id",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the app",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						app := &App{
							Id:         c.String("id"),
							Name:       c.String("name"),
							DataNodeId: c.String("datanode"),
							Note:       c.String("note"),
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
						response, err := sendCliCommand(node, cliAppUpdateCommand)
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the app",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := c.String("id")
						cliAppRemoveCommand := &Command{
							Type: "CLI_APP_REMOVE",
							Data: id,
						}
						response, err := sendCliCommand(node, cliAppRemoveCommand)
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
			Name:    "query",
			Aliases: []string{"q"},
			Usage:   "query commands",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "add a new query",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the query",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "script, s",
							Usage: "script of the query",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the query",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
						query := &Query{
							Id:     id,
							Name:   c.String("name"),
							AppId:  c.String("app"),
							Script: c.String("script"),
							Note:   c.String("note"),
						}
						queryJSONBytes, err := json.Marshal(query)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliQueryAddCommand := &Command{
							Type: "CLI_QUERY_ADD",
							Data: string(queryJSONBytes),
						}
						response, err := sendCliCommand(node, cliQueryAddCommand)
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
					Usage: "update an existing query",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the query",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the query",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "script, s",
							Usage: "script of the query",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the query",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						query := &Query{
							Id:     c.String("id"),
							Name:   c.String("name"),
							AppId:  c.String("app"),
							Script: c.String("script"),
							Note:   c.String("note"),
						}
						queryJSONBytes, err := json.Marshal(query)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliQueryUpdateCommand := &Command{
							Type: "CLI_QUERY_UPDATE",
							Data: string(queryJSONBytes),
						}
						response, err := sendCliCommand(node, cliQueryUpdateCommand)
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
					Usage: "remove an existing query",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, id",
							Usage: "id of the app",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						query := &Query{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						queryJSONBytes, err := json.Marshal(query)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliQueryRemoveCommand := &Command{
							Type: "CLI_QUERY_REMOVE",
							Data: string(queryJSONBytes),
						}
						response, err := sendCliCommand(node, cliQueryRemoveCommand)
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
			Name:    "job",
			Aliases: []string{"j"},
			Usage:   "job commands",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "add a new job",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the job",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "cron, c",
							Usage: "cron expression of the job",
						},
						cli.StringFlag{
							Name:  "script, s",
							Usage: "script of the job",
						},
						cli.StringFlag{
							Name:  "loopscript, l",
							Usage: "loop script of the job",
						},
						cli.BoolFlag{
							Name:  "auto, u",
							Usage: "auto start the job?",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the job",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
						job := &Job{
							Id:         id,
							Name:       c.String("name"),
							AppId:      c.String("app"),
							Script:     c.String("script"),
							LoopScript: c.String("loopscript"),
							Cron:       c.String("cron"),
							AutoStart:  c.Bool("auto"),
							Note:       c.String("note"),
						}
						jobJSONBytes, err := json.Marshal(job)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliJobAddCommand := &Command{
							Type: "CLI_JOB_ADD",
							Data: string(jobJSONBytes),
						}
						response, err := sendCliCommand(node, cliJobAddCommand)
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
					Usage: "update an existing job",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the job",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the job",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "cron, c",
							Usage: "cron expression of the job",
						},
						cli.StringFlag{
							Name:  "script, s",
							Usage: "script of the job",
						},
						cli.StringFlag{
							Name:  "loopscript, l",
							Usage: "loop script of the job",
						},
						cli.BoolFlag{
							Name:  "auto, u",
							Usage: "auto start the job?",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the job",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						job := &Job{
							Id:         c.String("id"),
							Name:       c.String("name"),
							AppId:      c.String("app"),
							Script:     c.String("script"),
							LoopScript: c.String("loopscript"),
							Cron:       c.String("cron"),
							AutoStart:  c.Bool("auto"),
							Note:       c.String("note"),
						}
						jobJSONBytes, err := json.Marshal(job)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliJobUpdateCommand := &Command{
							Type: "CLI_JOB_UPDATE",
							Data: string(jobJSONBytes),
						}
						response, err := sendCliCommand(node, cliJobUpdateCommand)
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
					Usage: "remove an existing job",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the job",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						job := &Job{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						jobJSONBytes, err := json.Marshal(job)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliJobRemoveCommand := &Command{
							Type: "CLI_JOB_REMOVE",
							Data: string(jobJSONBytes),
						}
						response, err := sendCliCommand(node, cliJobRemoveCommand)
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
					Name:  "start",
					Usage: "start a job",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the job",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						job := &Job{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						jobJSONBytes, err := json.Marshal(job)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliJobStartCommand := &Command{
							Type: "CLI_JOB_START",
							Data: string(jobJSONBytes),
						}
						response, err := sendCliCommand(node, cliJobStartCommand)
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
					Name:  "restart",
					Usage: "restart a job",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the job",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						job := &Job{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						jobJSONBytes, err := json.Marshal(job)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliJobRestartCommand := &Command{
							Type: "CLI_JOB_RESTART",
							Data: string(jobJSONBytes),
						}
						response, err := sendCliCommand(node, cliJobRestartCommand)
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
					Name:  "stop",
					Usage: "stop a job",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the job",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						job := &Job{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						jobJSONBytes, err := json.Marshal(job)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliJobStopCommand := &Command{
							Type: "CLI_JOB_STOP",
							Data: string(jobJSONBytes),
						}
						response, err := sendCliCommand(node, cliJobStopCommand)
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
			Name:    "token",
			Aliases: []string{"t"},
			Usage:   "token commands",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "add a new token",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the token",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "mode, o",
							Usage: "script of the token",
						},
						cli.StringFlag{
							Name:  "target, g",
							Usage: "target of the token",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "note for the token",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
						token := &Token{
							Id:     id,
							Name:   c.String("name"),
							AppId:  c.String("app"),
							Mode:   c.String("mode"),
							Target: c.String("target"),
							Note:   c.String("note"),
						}
						tokenJSONBytes, err := json.Marshal(token)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliTokenAddCommand := &Command{
							Type: "CLI_TOKEN_ADD",
							Data: string(tokenJSONBytes),
						}
						response, err := sendCliCommand(node, cliTokenAddCommand)
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
					Usage: "update an existing token",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the token",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the token",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "mode, o",
							Usage: "script of the token",
						},
						cli.StringFlag{
							Name:  "target, g",
							Usage: "target of the token",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "a note for the token",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						token := &Token{
							Id:     c.String("id"),
							Name:   c.String("name"),
							AppId:  c.String("app"),
							Mode:   c.String("mode"),
							Target: c.String("target"),
							Note:   c.String("note"),
						}
						tokenJSONBytes, err := json.Marshal(token)
						if err != nil {
							fmt.Println(err)
							return err
						}
						cliTokenAddCommand := &Command{
							Type: "CLI_TOKEN_ADD",
							Data: string(tokenJSONBytes),
						}
						response, err := sendCliCommand(node, cliTokenAddCommand)
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
					Usage: "remove an existing token",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the token",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						token := &Token{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						jobJSONBytes, err := json.Marshal(token)
						if err != nil {
							return err
						}
						cliTokenRemoveCommand := &Command{
							Type: "CLI_TOKEN_REMOVE",
							Data: string(jobJSONBytes),
						}
						response, err := sendCliCommand(node, cliTokenRemoveCommand)
						if err != nil {
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
			Name:  "li",
			Usage: "local interceptor commands",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "add a new local interceptor",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the local interceptor",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "target, g",
							Usage: "target of the local interceptor",
						},
						cli.StringFlag{
							Name:  "callback, c",
							Usage: "callback query name for the local interceptor",
						},
						cli.StringFlag{
							Name:  "type, k",
							Usage: "type of the local interceptor",
						},
						cli.StringFlag{
							Name:  "criteria, f",
							Usage: "criteria type of the local interceptor",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "note for the local interceptor",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
						li := &LocalInterceptor{
							Id:       id,
							Name:     c.String("name"),
							AppId:    c.String("app"),
							Target:   c.String("target"),
							Callback: c.String("callback"),
							Type:     c.String("type"),
							Criteria: c.String("criteria"),
							Note:     c.String("note"),
						}
						liJSONBytes, err := json.Marshal(li)
						if err != nil {
							return err
						}
						cliLiAddCommand := &Command{
							Type: "CLI_LI_ADD",
							Data: string(liJSONBytes),
						}
						response, err := sendCliCommand(node, cliLiAddCommand)
						if err != nil {
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
					Usage: "update an existing local interceptor",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the local interceptor",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the local interceptor",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "target, g",
							Usage: "target of the local interceptor",
						},
						cli.StringFlag{
							Name:  "callback, c",
							Usage: "callback query name for the local interceptor",
						},
						cli.StringFlag{
							Name:  "type, k",
							Usage: "type of the local interceptor",
						},
						cli.StringFlag{
							Name:  "criteria, f",
							Usage: "criteria type of the local interceptor",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "note for the local interceptor",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						li := &LocalInterceptor{
							Id:       c.String("id"),
							Name:     c.String("name"),
							AppId:    c.String("app"),
							Target:   c.String("target"),
							Callback: c.String("callback"),
							Type:     c.String("type"),
							Criteria: c.String("criteria"),
							Note:     c.String("note"),
						}
						liJSONBytes, err := json.Marshal(li)
						if err != nil {
							return err
						}
						cliLiUpdateCommand := &Command{
							Type: "CLI_LI_UPDATE",
							Data: string(liJSONBytes),
						}
						response, err := sendCliCommand(node, cliLiUpdateCommand)
						if err != nil {
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
					Usage: "remove an existing local interceptor",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "the id of the local interceptor",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app name",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						li := &LocalInterceptor{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						liJSONBytes, err := json.Marshal(li)
						if err != nil {
							return err
						}
						cliLiRemoveCommand := &Command{
							Type: "CLI_LI_REMOVE",
							Data: string(liJSONBytes),
						}
						response, err := sendCliCommand(node, cliLiRemoveCommand)
						if err != nil {
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
			Name:  "ri",
			Usage: "remote interceptor commands",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "add a new remote interceptor",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "target, g",
							Usage: "target of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "method, e",
							Value: "POST",
							Usage: "method for the remote interceptor",
						},
						cli.StringFlag{
							Name:  "url, u",
							Usage: "url for the remote interceptor",
						},
						cli.StringFlag{
							Name:  "callback, c",
							Usage: "callback query name for the remote interceptor",
						},
						cli.StringFlag{
							Name:  "type, k",
							Usage: "type of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "action, o",
							Usage: "action type of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "criteria, f",
							Usage: "criteria type of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "note for the remote interceptor",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
						ri := &RemoteInterceptor{
							Id:         id,
							Name:       c.String("name"),
							AppId:      c.String("app"),
							Target:     c.String("target"),
							Method:     c.String("method"),
							Url:        c.String("url"),
							Callback:   c.String("callback"),
							Type:       c.String("type"),
							ActionType: c.String("action"),
							Criteria:   c.String("criteria"),
							Note:       c.String("note"),
						}
						riJSONBytes, err := json.Marshal(ri)
						if err != nil {
							return err
						}
						cliRiAddCommand := &Command{
							Type: "CLI_RI_ADD",
							Data: string(riJSONBytes),
						}
						response, err := sendCliCommand(node, cliRiAddCommand)
						if err != nil {
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
					Usage: "update an existing remote interceptor",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "id of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "name, n",
							Usage: "name of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
						cli.StringFlag{
							Name:  "target, g",
							Usage: "target of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "method, e",
							Value: "POST",
							Usage: "method for the remote interceptor",
						},
						cli.StringFlag{
							Name:  "url, u",
							Usage: "url for the remote interceptor",
						},
						cli.StringFlag{
							Name:  "callback, c",
							Usage: "callback query name for the remote interceptor",
						},
						cli.StringFlag{
							Name:  "type, k",
							Usage: "type of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "action, i",
							Usage: "action type of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "criteria, f",
							Usage: "criteria type of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "note, t",
							Usage: "note for the remote interceptor",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						ri := &RemoteInterceptor{
							Id:         c.String("id"),
							Name:       c.String("name"),
							AppId:      c.String("app"),
							Target:     c.String("target"),
							Method:     c.String("method"),
							Url:        c.String("url"),
							Callback:   c.String("callback"),
							Type:       c.String("type"),
							ActionType: c.String("action"),
							Criteria:   c.String("criteria"),
							Note:       c.String("note"),
						}
						riJSONBytes, err := json.Marshal(ri)
						if err != nil {
							return err
						}
						cliRiUpdateCommand := &Command{
							Type: "CLI_RI_UPDATE",
							Data: string(riJSONBytes),
						}
						response, err := sendCliCommand(node, cliRiUpdateCommand)
						if err != nil {
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
					Usage: "remove an existing remote interceptor",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
						cli.StringFlag{
							Name:  "id, i",
							Usage: "the id of the remote interceptor",
						},
						cli.StringFlag{
							Name:  "app, a",
							Usage: "app id",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						ri := &RemoteInterceptor{
							Id:    c.String("id"),
							AppId: c.String("app"),
						}
						riJSONBytes, err := json.Marshal(ri)
						if err != nil {
							return err
						}
						cliRiRemoveCommand := &Command{
							Type: "CLI_RI_REMOVE",
							Data: string(riJSONBytes),
						}
						response, err := sendCliCommand(node, cliRiRemoveCommand)
						if err != nil {
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
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						cliShowMasterCommand := &Command{
							Type: "CLI_SHOW_MASTER",
						}
						response, err := sendCliCommand(node, cliShowMasterCommand)
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
					Name:  "slave",
					Usage: "show slave api nodes info",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "node, N",
							Value: "127.0.0.1:2015",
							Usage: "node url, format: host:port. 127.0.0.1:2015 if empty",
						},
					},
					Action: func(c *cli.Context) error {
						node := c.String("node")
						cliShowApiNodesCommand := &Command{
							Type: "CLI_SHOW_API_NODES",
						}
						response, err := sendCliCommand(node, cliShowApiNodesCommand)
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
	err = app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

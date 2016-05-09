package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/elgs/gorest2"
	"github.com/gorilla/websocket"
)

var wsConns = make(map[string]*websocket.Conn)
var masterData MasterData

func loadMasterData(file string)             {}
func storeMasterData(masterData *MasterData) {}

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
	}

	service := &CliService{
		EnableHttp: true,
		HostHttp:   "127.0.0.1",
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
					Action: func(c *cli.Context) {
						service.LoadConfigs(c)
						if len(strings.TrimSpace(service.SlaveOf)) > 0 {
							// load data from master if slave
							c, _, err := websocket.DefaultDialer.Dial("wss://"+service.SlaveOf+"/sys/ws", nil)
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
								Data: service.Id,
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

											if err := conn.WriteJSON(masterData); err != nil {
												fmt.Println(err)
											}
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
						serve(service)
						<-done
					},
				},
				{
					Name:  "stop",
					Usage: "stop service",
					Action: func(c *cli.Context) {
						if len(c.Args()) > 0 {
							_, err := http.Post(fmt.Sprint("http://127.0.0.1:", c.Args()[0], "/sys/shutdown"), "text/plain", nil)
							if err != nil {
								fmt.Println(err)
							}
						} else {
							fmt.Println("Usage: netdata service stop <shutdown_port>")
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
					Flags: []cli.Flag{
						cli.BoolTFlag{
							Name:  "full, f",
							Usage: "show a full list of data nodes",
						},
						cli.BoolTFlag{
							Name:  "compact, c",
							Usage: "show a compact list of data nodes",
						}},
					Action: func(c *cli.Context) {
						full := c.IsSet("full")
						compact := c.IsSet("compact")
						mode := 2
						if compact {
							mode = 0
						} else if full {
							mode = 1
						}
						masterData.ListDataNode(mode)
					},
				},
				{
					Name:  "add",
					Usage: "add a new data node",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "name, n",
							Usage: "",
						},
					},
					Action: func(c *cli.Context) {
						dataNode := &DataNode{
							Name: "x",
						}
						masterData.AddDataNode(dataNode)
					},
				},
				{
					Name:  "update",
					Usage: "add an existing data node",
					Action: func(c *cli.Context) {
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing data node",
					Action: func(c *cli.Context) {
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
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "add",
					Usage: "add a new api node",
					Action: func(c *cli.Context) {
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "update",
					Usage: "add an existing api node",
					Action: func(c *cli.Context) {
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing api node",
					Action: func(c *cli.Context) {
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
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "add",
					Usage: "add a new app",
					Action: func(c *cli.Context) {
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "update",
					Usage: "add an existing app",
					Action: func(c *cli.Context) {
						println("new task template: ", c.Args().First())
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing app",
					Action: func(c *cli.Context) {
						println("removed task template: ", c.Args().First())
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

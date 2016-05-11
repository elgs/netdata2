package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func processWsCommand(conn *websocket.Conn, message []byte) error {
	wsCommand := &Command{}
	json.Unmarshal(message, wsCommand)
	if wsCommand.Type == "WS_REGISTER" {
		wsConns[wsCommand.Data] = conn
	}
	return nil
}

func processCliCommand(message []byte) (string, error) {
	cliCommand := &Command{}
	json.Unmarshal(message, cliCommand)
	if cliCommand.Type == "CLI_DN_LIST" {
		return masterData.ListDataNode(cliCommand.Data), nil
	} else if cliCommand.Type == "CLI_DN_ADD" {
		dataNode := &DataNode{}
		err := json.Unmarshal([]byte(cliCommand.Data), dataNode)
		if err != nil {
			return "", err
		}
		err = masterData.AddDataNode(dataNode)
		if err != nil {
			return "", err
		}
	}
	return "", nil
}

func sendCliCommand(master string, command *Command) ([]byte, error) {
	message, err := json.Marshal(command)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("POST", "https://"+master+"/sys/cli", strings.NewReader(string(message)))
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return result, err
}

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
						if len(strings.TrimSpace(service.Master)) > 0 {
							// load data from master if slave
							c, _, err := websocket.DefaultDialer.Dial("wss://"+service.Master+"/sys/ws", nil)
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
									wsCommand := &Command{}
									json.Unmarshal(message, wsCommand)
								}
							}()

							regCommand := Command{
								Type: "WS_REGISTER",
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
										processWsCommand(conn, message)
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
							}
							result, err := processCliCommand(res)
							if err != nil {
								fmt.Fprint(w, err.Error())
							}
							fmt.Fprint(w, result)
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
					Action: func(c *cli.Context) {
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
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
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
					Action: func(c *cli.Context) {
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
						}
						cliDnAddCommand := &Command{
							Type: "CLI_DN_ADD",
							Data: string(dataNodeJSONBytes),
						}
						response, err := sendCliCommand(master, cliDnAddCommand)
						if err != nil {
							fmt.Println(err)
						}
						output := string(response)
						if output != "" {
							fmt.Println(strings.TrimSpace(output))
						}
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

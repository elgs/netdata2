// slave_func
package main

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

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

func RegisterToMaster(service *CliService, wsDrop chan bool) error {
	c, _, err := websocket.DefaultDialer.Dial("wss://"+service.Master+"/sys/ws", nil)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 5)
		wsDrop <- true
		return err
	}
	go func() {
		defer c.Close()
		defer func() { wsDrop <- true }()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Connection dropped. Reconnecting in 5 seconds...", err)
				time.Sleep(time.Second * 5)
				// Reconnect
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
		log.Println(err)
		time.Sleep(time.Second * 5)
		wsDrop <- true
		return err
	}
	log.Println("Connected to master:", service.Master)
	return nil
}

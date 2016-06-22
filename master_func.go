// master_func
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

func processWsCommandMaster(conn *websocket.Conn, message []byte) error {
	wsCommand := &Command{}
	json.Unmarshal(message, wsCommand)
	switch wsCommand.Type {
	case "WS_REGISTER":
		service := &CliService{}
		err := json.Unmarshal([]byte(wsCommand.Data), service)
		if err != nil {
			return err
		}

		apiNode := &ApiNode{
			Id:   service.Id,
			Name: conn.RemoteAddr().String(),
			//			ServerName: fmt.Sprint(service.HostHttps, ":", service.PortHttps),
		}
		err = masterData.AddApiNode(apiNode)
		if err != nil {
			conn.Close()
			return err
		}

		wsConns[service.Id] = conn
		log.Println(conn.RemoteAddr(), "connected.")
	}
	return nil
}

var masterDataMutex = &sync.Mutex{}

func propagateMasterData() error {
	masterDataMutex.Lock()
	var err error
	masterDataBytes, err := json.Marshal(masterData)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(service.DataFile, masterDataBytes, 0644)
	if err != nil {
		return err
	}
	masterDataMutex.Unlock()
	masterDataCommand := &Command{
		Type: "WS_MASTER_DATA",
		Data: string(masterDataBytes),
	}
	for _, conn := range wsConns {
		err = conn.WriteJSON(masterDataCommand)
		if err != nil {
			log.Println(err)
		}
	}
	return err
}

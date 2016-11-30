// master_func
package main

import (
	"encoding/json"
	"errors"
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
		slaveService := &CliService{}
		err := json.Unmarshal([]byte(wsCommand.Data), slaveService)
		if err != nil {
			conn.Close()
			return err
		}

		if slaveService.Secret != service.Secret {
			conn.WriteJSON("Failed to valid client secret.")
			return errors.New("Failed to valid client secret.")
		}

		apiNode := &ApiNode{
			Id:   slaveService.Id,
			Name: conn.RemoteAddr().String(),
		}
		err = AddApiNode(apiNode)
		if err != nil {
			conn.Close()
			return err
		}

		wsConns[slaveService.Id] = conn
		conn.WriteJSON("OK")
		log.Println(conn.RemoteAddr(), "connected.")
	}
	return nil
}

var masterDataMutex = &sync.Mutex{}

func (this *MasterData) Propagate() error {
	masterDataMutex.Lock()
	var err error
	masterDataBytes, err := json.Marshal(this)
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

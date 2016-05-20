// master_func
package main

import (
	"encoding/json"
	"log"

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
		wsConns[service.Id] = conn
		log.Println(conn.RemoteAddr(), "connected.")

		masterDataBytes, err := json.Marshal(masterData)
		if err != nil {
			return err
		}
		masterDataCommand := &Command{
			Type: "WS_MASTER_DATA",
			Data: string(masterDataBytes),
		}
		conn.WriteJSON(masterDataCommand)
	}
	return nil
}

func propagateMasterData() error {
	var err error
	masterDataBytes, err := json.Marshal(masterData)
	if err != nil {
		return err
	}
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

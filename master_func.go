// master_func
package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

func processCliCommand(message []byte) (string, error) {
	cliCommand := &Command{}
	json.Unmarshal(message, cliCommand)
	switch cliCommand.Type {
	case "CLI_DN_LIST":
		return masterData.ListDataNodes(cliCommand.Data), nil
	case "CLI_DN_ADD":
		dataNode := &DataNode{}
		err := json.Unmarshal([]byte(cliCommand.Data), dataNode)
		if err != nil {
			return "", err
		}
		err = masterData.AddDataNode(dataNode)
		if err != nil {
			return "", err
		}
	case "CLI_DN_UPDATE":
		dataNode := &DataNode{}
		err := json.Unmarshal([]byte(cliCommand.Data), dataNode)
		if err != nil {
			return "", err
		}
		err = masterData.UpdateDataNode(dataNode)
		if err != nil {
			return "", err
		}
	case "CLI_DN_REMOVE":
		err := masterData.RemoveDataNode(cliCommand.Data)
		if err != nil {
			return "", err
		}
	case "CLI_APP_LIST":
		return masterData.ListApps(cliCommand.Data), nil
	case "CLI_APP_ADD":
		app := &App{}
		err := json.Unmarshal([]byte(cliCommand.Data), app)
		if err != nil {
			return "", err
		}
		err = masterData.AddApp(app)
		if err != nil {
			return "", err
		}
	case "CLI_APP_UPDATE":
		app := &App{}
		err := json.Unmarshal([]byte(cliCommand.Data), app)
		if err != nil {
			return "", err
		}
		err = masterData.UpdateApp(app)
		if err != nil {
			return "", err
		}
	case "CLI_APP_REMOVE":
		err := masterData.RemoveApp(cliCommand.Data)
		if err != nil {
			return "", err
		}
	case "CLI_SHOW_MASTER":
		masterDataBytes, err := json.Marshal(masterData)
		if err != nil {
			return "", err
		}
		return string(masterDataBytes), nil
	}
	return "", nil
}

func processWsCommandMaster(conn *websocket.Conn, message []byte) error {
	wsCommand := &Command{}
	json.Unmarshal(message, wsCommand)
	switch wsCommand.Type {
	case "WS_REGISTER":
		wsConns[wsCommand.Data] = conn
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

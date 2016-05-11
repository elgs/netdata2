// master_data
package main

import (
	"bytes"
	"errors"
	"fmt"
)

type Command struct {
	Type string
	Data string
	Meta map[string]interface{}
}

type MasterData struct {
	Version            int64
	DataNodes          []DataNode
	ApiNodes           []ApiNode
	Apps               []App
	Queries            []Query
	Jobs               []Job
	Tokens             []Token
	LocalInterceptors  []LocalInterceptor
	RemoteInterceptors []RemoteInterceptor
}

type DataNode struct {
	Name     string
	Username string
	Password string
	Host     string
	Port     int
	Type     string
	Note     string
	Status   string
}
type ApiNode struct {
	Name        string
	ServerName  string
	ServerIP4   string
	ServerIP6   string
	ServerPort  int64
	CountryCode string
	Region      string
	SuperRegion string
	Note        string
	Status      string
}
type App struct {
	Name         string
	DbName       string
	DataNodeName string
	Note         string
	Status       string
}
type Query struct {
	Name    string
	Script  string
	AppName string
	Note    string
	Status  string
}
type Job struct {
	Name       string
	Cron       string
	Mode       string
	Script     string
	LoopScript string
	AppName    string
	Note       string
	Status     string
}
type Token struct {
	Name    string
	Token   string
	Mode    string
	Targets string
	AppName string
	Note    string
	Status  string
}
type LocalInterceptor struct {
	AppName    string
	Target     string
	Callback   string
	Type       string
	ActionType string
	Criteria   string
	Note       string
	Status     string
}
type RemoteInterceptor struct {
	AppName    string
	Target     string
	Method     string
	Url        string
	Type       string
	ActionType string
	Criteria   string
	Callback   string
	Note       string
	Status     string
}

func (this *MasterData) AddDataNode(dataNode *DataNode) error {
	for _, v := range this.DataNodes {
		if v.Name == dataNode.Name {
			return errors.New("Data node existed: " + dataNode.Name)
		}
	}
	this.DataNodes = append(this.DataNodes, *dataNode)
	this.Version++
	return nil
}
func (this *MasterData) RemoveDataNode(name string) error {
	index := -1
	for i, v := range this.DataNodes {
		if v.Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Data node not found: " + name)
	}
	this.DataNodes = append(this.DataNodes[:index], this.DataNodes[index+1:]...)
	return nil
}
func (this *MasterData) UpdateDataNode(dataNode *DataNode) error {
	index := -1
	for i, v := range this.DataNodes {
		if v.Name == dataNode.Name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Data node not found: " + dataNode.Name)
	}
	this.DataNodes = append(this.DataNodes[:index], *dataNode)
	this.DataNodes = append(this.DataNodes, this.DataNodes[index+1:]...)
	return nil
}
func (this *MasterData) ListDataNode(mode string) string {
	var buffer bytes.Buffer
	for _, dataNode := range masterData.DataNodes {
		if mode == "compact" {
			buffer.WriteString(dataNode.Name + " ")
		} else if mode == "full" {
			buffer.WriteString(fmt.Sprintln(dataNode.Name, dataNode.Host))
		} else {
			buffer.WriteString(dataNode.Name + "\n")
		}
	}
	return buffer.String()
}

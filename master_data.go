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
	Id       string
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
	Id          string
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
	Id         string
	Name       string
	DbName     string
	DataNodeId string
	Note       string
	Status     string
}
type Query struct {
	Id     string
	Name   string
	Script string
	AppId  string
	Note   string
	Status string
}
type Job struct {
	Id   string
	Name string
	Cron string
	//	Mode       string
	Script     string
	LoopScript string
	AppId      string
	Note       string
	Status     string
}
type Token struct {
	Id     string
	Name   string
	Token  string
	Mode   string
	Target string
	AppId  string
	Note   string
	Status string
}
type LocalInterceptor struct {
	Id         string
	Name       string
	AppId      string
	Target     string
	Callback   string
	Type       string
	ActionType string
	Criteria   string
	Note       string
	Status     string
}
type RemoteInterceptor struct {
	Id         string
	Name       string
	AppId      string
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
	return propagateMasterData()
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
	this.Version++
	return propagateMasterData()
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
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) ListDataNodes(mode string) string {
	var buffer bytes.Buffer
	for _, dataNode := range this.DataNodes {
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

func (this *MasterData) AddApp(app *App) error {
	for _, v := range this.Apps {
		if v.Name == app.Name {
			return errors.New("App existed: " + app.Name)
		}
	}
	dataNodeFound := false
	for _, v := range this.DataNodes {
		if v.Name == app.DataNodeId {
			dataNodeFound = true
			break
		}
	}
	if !dataNodeFound {
		return errors.New("Data node does not exist: " + app.DataNodeId)
	}
	this.Apps = append(this.Apps, *app)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveApp(name string) error {
	index := -1
	for i, v := range this.Apps {
		if v.Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("App not found: " + name)
	}
	this.Apps = append(this.Apps[:index], this.Apps[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateApp(app *App) error {
	index := -1
	for i, v := range this.Apps {
		if v.Name == app.Name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("App not found: " + app.Name)
	}

	dataNodeFound := false
	for _, v := range this.DataNodes {
		if v.Name == app.DataNodeId {
			dataNodeFound = true
			break
		}
	}
	if !dataNodeFound {
		return errors.New("Data node does not exist: " + app.DataNodeId)
	}

	this.Apps = append(this.Apps[:index], *app)
	this.Apps = append(this.Apps, this.Apps[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) ListApps(mode string) string {
	var buffer bytes.Buffer
	for _, app := range this.Apps {
		if mode == "compact" {
			buffer.WriteString(app.Name + " ")
		} else if mode == "full" {
			buffer.WriteString(fmt.Sprintln(app.Name, app.DataNodeId))
		} else {
			buffer.WriteString(app.Name + "\n")
		}
	}
	return buffer.String()
}

func (this *MasterData) AddApiNode(apiNode *ApiNode) error {
	for _, v := range this.ApiNodes {
		if v.Name == apiNode.Name {
			return errors.New("API node existed: " + apiNode.Name)
		}
	}
	this.ApiNodes = append(this.ApiNodes, *apiNode)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) RemoveApiNode(name string) error {
	index := -1
	for i, v := range this.ApiNodes {
		if v.Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("API node not found: " + name)
	}
	this.ApiNodes = append(this.ApiNodes[:index], this.ApiNodes[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddQuery(query *Query) error {
	for _, v := range this.Queries {
		if v.Name == query.Name {
			return errors.New("Query existed: " + query.Name)
		}
	}
	appFound := false
	for _, v := range this.Apps {
		if v.Name == query.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + query.AppId)
	}
	this.Queries = append(this.Queries, *query)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveQuery(name string, appName string) error {
	index := -1
	for i, v := range this.Queries {
		if v.Name == name && v.AppId == appName {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Query not found: " + name)
	}
	this.Queries = append(this.Queries[:index], this.Queries[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateQuery(query *Query) error {
	index := -1
	for i, v := range this.Queries {
		if v.Name == query.Name && v.AppId == query.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Query not found: " + query.Name)
	}

	appFound := false
	for _, v := range this.Apps {
		if v.Name == query.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + query.AppId)
	}

	this.Queries = append(this.Queries[:index], *query)
	this.Queries = append(this.Queries, this.Queries[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddJob(job *Job) error {
	for _, v := range this.Jobs {
		if v.Name == job.Name {
			return errors.New("Job existed: " + job.Name)
		}
	}
	appFound := false
	for _, v := range this.Apps {
		if v.Name == job.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + job.AppId)
	}
	this.Jobs = append(this.Jobs, *job)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveJob(name string, appName string) error {
	index := -1
	for i, v := range this.Jobs {
		if v.Name == name && v.AppId == appName {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Job not found: " + name)
	}
	this.Jobs = append(this.Jobs[:index], this.Jobs[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateJob(job *Job) error {
	index := -1
	for i, v := range this.Jobs {
		if v.Name == job.Name && v.AppId == job.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Job not found: " + job.Name)
	}

	appFound := false
	for _, v := range this.Apps {
		if v.Name == job.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + job.AppId)
	}

	this.Jobs = append(this.Jobs[:index], *job)
	this.Jobs = append(this.Jobs, this.Jobs[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddToken(token *Token) error {
	for _, v := range this.Tokens {
		if v.Name == token.Name || v.Token == token.Token {
			return errors.New("token existed: " + token.Name + " - " + token.Token)
		}
	}
	appFound := false
	for _, v := range this.Apps {
		if v.Name == token.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + token.AppId)
	}
	this.Tokens = append(this.Tokens, *token)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveToken(token string, appName string) error {
	index := -1
	for i, v := range this.Tokens {
		if v.Token == token && v.AppId == appName {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Token not found: " + token)
	}
	this.Tokens = append(this.Tokens[:index], this.Tokens[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateToken(token *Token) error {
	index := -1
	for i, v := range this.Tokens {
		if v.Token == token.Token && v.AppId == token.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Token not found: " + token.Token)
	}

	appFound := false
	for _, v := range this.Apps {
		if v.Name == token.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + token.AppId)
	}

	this.Tokens = append(this.Tokens[:index], *token)
	this.Tokens = append(this.Tokens, this.Tokens[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddLI(li *LocalInterceptor) error {
	for _, v := range this.LocalInterceptors {
		if v.Name == li.Name {
			return errors.New("Local interceptor existed: " + li.Name)
		}
	}
	appFound := false
	for _, v := range this.Apps {
		if v.Name == li.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + li.AppId)
	}
	this.LocalInterceptors = append(this.LocalInterceptors, *li)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveLI(name string, appName string) error {
	index := -1
	for i, v := range this.LocalInterceptors {
		if v.Name == name && v.AppId == appName {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Local interceptor not found: " + name)
	}
	this.LocalInterceptors = append(this.LocalInterceptors[:index], this.LocalInterceptors[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateLI(li *LocalInterceptor) error {
	index := -1
	for i, v := range this.LocalInterceptors {
		if v.Name == li.Name && v.AppId == li.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Local interceptor not found: " + li.Name)
	}

	appFound := false
	for _, v := range this.LocalInterceptors {
		if v.Name == li.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + li.AppId)
	}

	this.LocalInterceptors = append(this.LocalInterceptors[:index], *li)
	this.LocalInterceptors = append(this.LocalInterceptors, this.LocalInterceptors[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddRI(ri *RemoteInterceptor) error {
	for _, v := range this.RemoteInterceptors {
		if v.Name == ri.Name {
			return errors.New("Remote interceptor existed: " + ri.Name)
		}
	}
	appFound := false
	for _, v := range this.Apps {
		if v.Name == ri.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + ri.AppId)
	}
	this.RemoteInterceptors = append(this.RemoteInterceptors, *ri)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveRI(name string, appName string) error {
	index := -1
	for i, v := range this.RemoteInterceptors {
		if v.Name == name && v.AppId == appName {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Remote interceptor not found: " + name)
	}
	this.RemoteInterceptors = append(this.RemoteInterceptors[:index], this.RemoteInterceptors[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateRI(ri *RemoteInterceptor) error {
	index := -1
	for i, v := range this.RemoteInterceptors {
		if v.Name == ri.Name && v.AppId == ri.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Remote interceptor not found: " + ri.Name)
	}

	appFound := false
	for _, v := range this.RemoteInterceptors {
		if v.Name == ri.AppId {
			appFound = true
			break
		}
	}
	if !appFound {
		return errors.New("App does not exist: " + ri.AppId)
	}

	this.RemoteInterceptors = append(this.RemoteInterceptors[:index], *ri)
	this.RemoteInterceptors = append(this.RemoteInterceptors, this.RemoteInterceptors[index+1:]...)
	this.Version++
	return propagateMasterData()
}

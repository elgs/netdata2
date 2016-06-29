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
type App struct {
	Id         string
	Name       string
	DbName     string
	DataNodeId string
	DataNode   *DataNode
	Note       string
	Status     string
}
type Query struct {
	Id     string
	Name   string
	Script string
	AppId  string
	App    *App
	Note   string
	Status string
}
type Job struct {
	Id         string
	Name       string
	Cron       string
	Script     string
	LoopScript string
	AppId      string
	App        *App
	Note       string
	Status     string
}
type Token struct {
	Id     string
	Name   string
	Mode   string
	Target string
	AppId  string
	App    *App
	Note   string
	Status string
}
type LocalInterceptor struct {
	Id         string
	Name       string
	AppId      string
	App        *App
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
	App        *App
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
func (this *MasterData) RemoveDataNode(id string) error {
	index := -1
	for i, v := range this.DataNodes {
		if v.Id == id {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Data node not found: " + id)
	}
	this.DataNodes = append(this.DataNodes[:index], this.DataNodes[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateDataNode(dataNode *DataNode) error {
	index := -1
	for i, v := range this.DataNodes {
		if v.Id == dataNode.Id {
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
	for _, v := range this.DataNodes {
		if v.Id == app.DataNodeId {
			app.DataNode = &v
			break
		}
	}
	if app.DataNode == nil {
		return errors.New("Data node does not exist: " + app.DataNodeId)
	}
	err := OnAppCreateOrUpdate(app)
	if err != nil {
		return err
	}
	this.Apps = append(this.Apps, *app)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveApp(id string) error {
	index := -1
	for i, v := range this.Apps {
		if v.Id == id {
			index = i
			err := OnAppRemove(&v)
			if err != nil {
				return err
			}
			break
		}
	}
	if index == -1 {
		return errors.New("App not found: " + id)
	}
	this.Apps = append(this.Apps[:index], this.Apps[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateApp(app *App) error {
	index := -1
	for i, v := range this.Apps {
		if v.Id == app.Id {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("App not found: " + app.Name)
	}

	for _, v := range this.DataNodes {
		if v.Id == app.DataNodeId {
			app.DataNode = &v
			break
		}
	}
	if app.DataNode == nil {
		return errors.New("Data node does not exist: " + app.DataNodeId)
	}

	OnAppCreateOrUpdate(app)
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

func (this *MasterData) AddQuery(query *Query) error {
	for _, v := range this.Queries {
		if v.Name == query.Name && v.Id == query.AppId {
			return errors.New("Query existed: " + query.Name)
		}
	}
	for _, v := range this.Apps {
		if v.Id == query.AppId {
			query.App = &v
			break
		}
	}
	if query.App == nil {
		return errors.New("App does not exist: " + query.AppId)
	}
	this.Queries = append(this.Queries, *query)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveQuery(id string, appId string) error {
	index := -1
	for i, v := range this.Queries {
		if v.Id == id && v.AppId == appId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Query not found: " + id)
	}
	this.Queries = append(this.Queries[:index], this.Queries[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateQuery(query *Query) error {
	index := -1
	for i, v := range this.Queries {
		if v.Id == query.Id && v.AppId == query.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Query not found: " + query.Name)
	}

	for _, v := range this.Apps {
		if v.Id == query.AppId {
			query.App = &v
			break
		}
	}
	if query.App == nil {
		return errors.New("App does not exist: " + query.AppId)
	}

	this.Queries = append(this.Queries[:index], *query)
	this.Queries = append(this.Queries, this.Queries[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddJob(job *Job) error {
	for _, v := range this.Jobs {
		if v.Name == job.Name && v.AppId == job.AppId {
			return errors.New("Job existed: " + job.Name)
		}
	}
	for _, v := range this.Apps {
		if v.Id == job.AppId {
			job.App = &v
			break
		}
	}
	if job.App == nil {
		return errors.New("App does not exist: " + job.AppId)
	}
	this.Jobs = append(this.Jobs, *job)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveJob(id string, appId string) error {
	index := -1
	for i, v := range this.Jobs {
		if v.Id == id && v.AppId == appId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Job not found: " + id)
	}
	this.Jobs = append(this.Jobs[:index], this.Jobs[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateJob(job *Job) error {
	index := -1
	for i, v := range this.Jobs {
		if v.Id == job.Id && v.AppId == job.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Job not found: " + job.Name)
	}

	for _, v := range this.Apps {
		if v.Id == job.AppId {
			job.App = &v
			break
		}
	}
	if job.App == nil {
		return errors.New("App does not exist: " + job.AppId)
	}

	this.Jobs = append(this.Jobs[:index], *job)
	this.Jobs = append(this.Jobs, this.Jobs[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddToken(token *Token) error {
	for _, v := range this.Tokens {
		if v.Name == token.Name || v.Id == token.Id {
			return errors.New("token existed: " + token.Name + " - " + token.Id)
		}
	}
	for _, v := range this.Apps {
		if v.Id == token.AppId {
			token.App = &v
			break
		}
	}
	if token.App == nil {
		return errors.New("App does not exist: " + token.AppId)
	}
	this.Tokens = append(this.Tokens, *token)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveToken(id string, appId string) error {
	index := -1
	for i, v := range this.Tokens {
		if v.Id == id && v.AppId == appId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Token not found: " + id)
	}
	this.Tokens = append(this.Tokens[:index], this.Tokens[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateToken(token *Token) error {
	index := -1
	for i, v := range this.Tokens {
		if v.Id == token.Id && v.AppId == token.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Token not found: " + token.Id)
	}

	for _, v := range this.Apps {
		if v.Id == token.AppId {
			token.App = &v
			break
		}
	}
	if token.App == nil {
		return errors.New("App does not exist: " + token.AppId)
	}

	this.Tokens = append(this.Tokens[:index], *token)
	this.Tokens = append(this.Tokens, this.Tokens[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddLI(li *LocalInterceptor) error {
	for _, v := range this.LocalInterceptors {
		if v.Name == li.Name && v.AppId == li.AppId {
			return errors.New("Local interceptor existed: " + li.Name)
		}
	}
	for _, v := range this.Apps {
		if v.Id == li.AppId {
			li.App = &v
			break
		}
	}
	if li.App == nil {
		return errors.New("App does not exist: " + li.AppId)
	}
	this.LocalInterceptors = append(this.LocalInterceptors, *li)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveLI(id string, appId string) error {
	index := -1
	for i, v := range this.LocalInterceptors {
		if v.Id == id && v.AppId == appId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Local interceptor not found: " + id)
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

	for _, v := range this.Apps {
		if v.Id == li.AppId {
			li.App = &v
			break
		}
	}
	if li.App == nil {
		return errors.New("App does not exist: " + li.AppId)
	}

	this.LocalInterceptors = append(this.LocalInterceptors[:index], *li)
	this.LocalInterceptors = append(this.LocalInterceptors, this.LocalInterceptors[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func (this *MasterData) AddRI(ri *RemoteInterceptor) error {
	for _, v := range this.RemoteInterceptors {
		if v.Name == ri.Name && v.AppId == ri.AppId {
			return errors.New("Remote interceptor existed: " + ri.Name)
		}
	}
	for _, v := range this.Apps {
		if v.Id == ri.AppId {
			ri.App = &v
			break
		}
	}
	if ri.App == nil {
		return errors.New("App does not exist: " + ri.AppId)
	}
	this.RemoteInterceptors = append(this.RemoteInterceptors, *ri)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) RemoveRI(id string, appId string) error {
	index := -1
	for i, v := range this.RemoteInterceptors {
		if v.Id == id && v.AppId == appId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Remote interceptor not found: " + id)
	}
	this.RemoteInterceptors = append(this.RemoteInterceptors[:index], this.RemoteInterceptors[index+1:]...)
	this.Version++
	return propagateMasterData()
}
func (this *MasterData) UpdateRI(ri *RemoteInterceptor) error {
	index := -1
	for i, v := range this.RemoteInterceptors {
		if v.Id == ri.Id && v.AppId == ri.AppId {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("Remote interceptor not found: " + ri.Name)
	}

	for _, v := range this.Apps {
		if v.Id == ri.AppId {
			ri.App = &v
			break
		}
	}
	if ri.App == nil {
		return errors.New("App does not exist: " + ri.AppId)
	}

	this.RemoteInterceptors = append(this.RemoteInterceptors[:index], *ri)
	this.RemoteInterceptors = append(this.RemoteInterceptors, this.RemoteInterceptors[index+1:]...)
	this.Version++
	return propagateMasterData()
}

func AddApiNode(apiNode *ApiNode) error {
	for _, v := range apiNodes {
		if v.Name == apiNode.Name {
			return errors.New("API node existed: " + apiNode.Name)
		}
	}
	apiNodes = append(apiNodes, *apiNode)
	return nil
}

func RemoveApiNode(remoteAddr string) error {
	index := -1
	for i, v := range apiNodes {
		if v.Name == remoteAddr {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("API node not found: " + remoteAddr)
	}
	apiNodes = append(apiNodes[:index], apiNodes[index+1:]...)
	return nil
}

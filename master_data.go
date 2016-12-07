// master_data
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Command struct {
	Type   string
	Data   string
	Secret string
	Meta   map[string]interface{}
}

type MasterData struct {
	Version   int64
	DataNodes []*DataNode
	Apps      []*App
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
	Id                 string
	Name               string
	DbName             string
	DataNodeId         string
	Note               string
	Status             string
	Queries            []*Query
	Jobs               []*Job
	Tokens             []*Token
	LocalInterceptors  []*LocalInterceptor
	RemoteInterceptors []*RemoteInterceptor
}
type Query struct {
	Id         string
	Name       string
	ScriptPath string
	ScriptText string
	AppId      string
	Note       string
	Status     string
}
type Job struct {
	Id             string
	Name           string
	Cron           string
	ScriptPath     string
	ScriptText     string
	AutoStart      int
	LoopScriptPath string
	LoopScriptText string
	AppId          string
	Note           string
	Status         string
}
type Token struct {
	Id     string
	Name   string
	Mode   string
	Target string
	AppId  string
	Note   string
	Status string
}
type LocalInterceptor struct {
	Id       string
	Name     string
	AppId    string
	Target   string
	Callback string
	Type     string
	Note     string
	Status   string
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
	this.DataNodes = append(this.DataNodes, dataNode)
	this.Version++
	return masterData.Propagate()
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
	copy(this.DataNodes[index:], this.DataNodes[index+1:])
	this.DataNodes[len(this.DataNodes)-1] = nil
	this.DataNodes = this.DataNodes[:len(this.DataNodes)-1]
	//	this.DataNodes = append(this.DataNodes[:index], this.DataNodes[index+1:]...)
	this.Version++
	return masterData.Propagate()
}
func (this *MasterData) UpdateDataNode(dataNode *DataNode) error {
	for i, v := range this.DataNodes {
		if v.Id == dataNode.Id {
			if v.Name != "__not_set__" {
				v.Name = dataNode.Name
			}
			if v.Host != "__not_set__" {
				v.Host = dataNode.Host
			}
			if v.Port != -1 {
				v.Port = dataNode.Port
			}
			if v.Username != "__not_set__" {
				v.Username = dataNode.Username
			}
			if v.Password != "__not_set__" {
				v.Password = dataNode.Password
			}
			if v.Note != "__not_set__" {
				v.Note = dataNode.Note
			}
			this.DataNodes[i] = v
			this.Version++
			return masterData.Propagate()
		}
	}
	return errors.New("Data node not found: " + dataNode.Name)
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
	found := false
	for _, v := range this.DataNodes {
		if v.Id == app.DataNodeId {
			found = true
			break
		}
	}
	if !found {
		return errors.New("Data node does not exist: " + app.DataNodeId)
	}
	err := app.OnAppCreateOrUpdate()
	if err != nil {
		return err
	}
	this.Apps = append(this.Apps, app)
	this.Version++
	return masterData.Propagate()
}
func (this *MasterData) RemoveApp(id string) error {
	index := -1
	for i, v := range this.Apps {
		if v.Id == id {
			index = i
			err := v.OnAppRemove()
			if err != nil {
				return err
			}
			break
		}
	}
	if index == -1 {
		return errors.New("App not found: " + id)
	}
	copy(this.Apps[index:], this.Apps[index+1:])
	this.Apps[len(this.Apps)-1] = nil
	this.Apps = this.Apps[:len(this.Apps)-1]
	//	this.Apps = append(this.Apps[:index], this.Apps[index+1:]...)
	this.Version++
	return masterData.Propagate()
}
func (this *MasterData) UpdateApp(app *App) error {
	iApp := -1
	var vApp *App
	for i, v := range this.Apps {
		if v.Id == app.Id {
			iApp = i
			vApp = v
			break
		}
	}
	if iApp == -1 {
		return errors.New("App not found: " + app.Name)
	}

	found := false
	for _, v := range this.DataNodes {
		if v.Id == app.DataNodeId {
			found = true
			break
		}
	}
	if !found {
		return errors.New("Data node does not exist: " + app.DataNodeId)
	}

	if app.Name != "__not_set__" {
		vApp.Name = app.Name
	}
	if app.DataNodeId != "__not_set__" {
		vApp.DataNodeId = app.DataNodeId
	}
	if app.Note != "__not_set__" {
		vApp.Note = app.Note
	}
	vApp.OnAppCreateOrUpdate()
	this.Apps[iApp] = vApp
	this.Version++
	return masterData.Propagate()
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
	for iApp, vApp := range this.Apps {
		if vApp.Id == query.AppId {
			for _, vQuery := range this.Apps[iApp].Queries {
				if vQuery.Name == query.Name && vQuery.AppId == query.AppId {
					return errors.New("Query existed: " + query.Name)
				}
			}
			this.Apps[iApp].Queries = append(this.Apps[iApp].Queries, query)
			err := query.Reload()
			if err != nil {
				return err
			}
			this.Version++
			return masterData.Propagate()
		}
	}
	return errors.New("App does not exist: " + query.AppId)
}
func (this *MasterData) RemoveQuery(id string, appId string) error {
	for iApp, _ := range this.Apps {
		if this.Apps[iApp].Id == appId {
			for iQuery, vQuery := range this.Apps[iApp].Queries {
				if vQuery.Id == id && vQuery.AppId == appId {
					copy(this.Apps[iApp].Queries[iQuery:], this.Apps[iApp].Queries[iQuery+1:])
					this.Apps[iApp].Queries[len(this.Apps[iApp].Queries)-1] = nil
					this.Apps[iApp].Queries = this.Apps[iApp].Queries[:len(this.Apps[iApp].Queries)-1]
					//					this.Apps[iApp].Queries = append(this.Apps[iApp].Queries[:iQuery], this.Apps[iApp].Queries[iQuery+1:]...)
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Query not found: " + id)
}
func (this *MasterData) UpdateQuery(query *Query) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == query.AppId {
			for iQuery, vQuery := range this.Apps[iApp].Queries {
				if vQuery.Id == query.Id && vQuery.AppId == query.AppId {
					if query.Name != "__not_set__" {
						vQuery.Name = query.Name
					}
					if query.ScriptPath != "__not_set__" {
						vQuery.ScriptPath = query.ScriptPath
					}
					if query.Note != "__not_set__" {
						vQuery.Note = query.Note
					}
					this.Apps[iApp].Queries[iQuery] = vQuery
					err := vQuery.Reload()
					if err != nil {
						return err
					}
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Query not found: " + query.Name)
}
func (this *MasterData) ReloadAllQueries(appId string) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == appId {
			for iQuery, vQuery := range this.Apps[iApp].Queries {
				if vQuery.AppId == appId {
					this.Apps[iApp].Queries[iQuery] = vQuery
					err := vQuery.Reload()
					if err != nil {
						return err
					}
				}
			}
			this.Version++
			return masterData.Propagate()
		}
	}
	return nil
}

func (this *MasterData) AddJob(job *Job) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == job.AppId {
			for _, vJob := range this.Apps[iApp].Jobs {
				if vJob.Name == job.Name && vJob.AppId == job.AppId {
					return errors.New("Job existed: " + job.Name)
				}
			}
			if job.AutoStart == 1 {
				job.Start()
			}
			this.Apps[iApp].Jobs = append(this.Apps[iApp].Jobs, job)
			this.Version++
			return masterData.Propagate()
		}
	}
	return errors.New("App does not exist: " + job.AppId)
}
func (this *MasterData) RemoveJob(id string, appId string) error {
	for iApp, _ := range this.Apps {
		if this.Apps[iApp].Id == appId {
			for iJob, vJob := range this.Apps[iApp].Jobs {
				if vJob.Id == id && vJob.AppId == appId {
					if vJob.Started() {
						vJob.Stop()
					}
					copy(this.Apps[iApp].Jobs[iJob:], this.Apps[iApp].Jobs[iJob+1:])
					this.Apps[iApp].Jobs[len(this.Apps[iApp].Jobs)-1] = nil
					this.Apps[iApp].Jobs = this.Apps[iApp].Jobs[:len(this.Apps[iApp].Jobs)-1]
					//					this.Apps[iApp].Jobs = append(this.Apps[iApp].Jobs[:iJob], this.Apps[iApp].Jobs[iJob+1:]...)
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Job not found: " + id)
}
func (this *MasterData) UpdateJob(job *Job) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == job.AppId {
			for iJob, vJob := range this.Apps[iApp].Jobs {
				if vJob.Id == job.Id && vJob.AppId == job.AppId {
					if job.Name != "__not_set__" {
						vJob.Name = job.Name
					}
					if job.ScriptPath != "__not_set__" {
						vJob.ScriptPath = job.ScriptPath
					}
					if job.LoopScriptPath != "__not_set__" {
						vJob.LoopScriptPath = job.LoopScriptPath
					}
					if job.Cron != "__not_set__" {
						vJob.Cron = job.Cron
					}
					if job.AutoStart != -1 {
						vJob.AutoStart = job.AutoStart
					}
					if job.Note != "__not_set__" {
						vJob.Note = job.Note
					}

					if vJob.Started() {
						job.Restart()
					}
					this.Apps[iApp].Jobs[iJob] = vJob
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Job not found: " + job.Name)
}

func (this *MasterData) StartJob(job *Job) error {
	for iApp, _ := range this.Apps {
		if this.Apps[iApp].Id == job.AppId {
			for _, vJob := range this.Apps[iApp].Jobs {
				if vJob.Id == job.Id {
					return vJob.Start()
				}
			}
		}
	}
	return errors.New("Job not found: " + job.Id)
}
func (this *MasterData) RestartJob(job *Job) error {
	err := this.StopJob(job)
	if err != nil {
		return err
	}
	return this.StartJob(job)
}
func (this *MasterData) StopJob(job *Job) error {
	return job.Stop()
}

func (this *MasterData) AddToken(token *Token) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == token.AppId {
			for _, vToken := range vApp.Tokens {
				if vToken.Name == token.Name && vToken.AppId == token.AppId {
					return errors.New("Token existed: " + token.Name)
				}
			}
			this.Apps[iApp].Tokens = append(this.Apps[iApp].Tokens, token)
			this.Version++
			return masterData.Propagate()
		}
	}
	return errors.New("App does not exist: " + token.AppId)
}
func (this *MasterData) RemoveToken(id string, appId string) error {
	for iApp, _ := range this.Apps {
		if this.Apps[iApp].Id == appId {
			for iToken, vToken := range this.Apps[iApp].Tokens {
				if vToken.Id == id && vToken.AppId == appId {
					copy(this.Apps[iApp].Tokens[iToken:], this.Apps[iApp].Tokens[iToken+1:])
					this.Apps[iApp].Tokens[len(this.Apps[iApp].Tokens)-1] = nil
					this.Apps[iApp].Tokens = this.Apps[iApp].Tokens[:len(this.Apps[iApp].Tokens)-1]
					//					this.Apps[iApp].Tokens = append(this.Apps[iApp].Tokens[:iToken], this.Apps[iApp].Tokens[iToken+1:]...)
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Token not found: " + id)
}
func (this *MasterData) UpdateToken(token *Token) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == token.AppId {
			for iToken, vToken := range this.Apps[iApp].Tokens {
				if vToken.Id == token.Id && vToken.AppId == token.AppId {
					if token.Name != "__not_set__" {
						vToken.Name = token.Name
					}
					if token.Mode != "__not_set__" {
						vToken.Mode = token.Mode
					}
					if token.Target != "__not_set__" {
						vToken.Target = token.Target
					}
					if token.Note != "__not_set__" {
						vToken.Note = token.Note
					}

					this.Apps[iApp].Tokens[iToken] = vToken
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Token not found: " + token.Name)
}

func (this *MasterData) AddLI(li *LocalInterceptor) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == li.AppId {
			for _, vLi := range vApp.LocalInterceptors {
				if vLi.Name == li.Name && vLi.AppId == li.AppId {
					return errors.New("Local interceptor existed: " + li.Name)
				}
			}
			this.Apps[iApp].LocalInterceptors = append(this.Apps[iApp].LocalInterceptors, li)
			this.Version++
			return masterData.Propagate()
		}
	}
	return errors.New("App does not exist: " + li.AppId)
}
func (this *MasterData) RemoveLI(id string, appId string) error {
	for iApp, _ := range this.Apps {
		if this.Apps[iApp].Id == appId {
			for iLi, vLi := range this.Apps[iApp].LocalInterceptors {
				if vLi.Id == id && vLi.AppId == appId {
					copy(this.Apps[iApp].LocalInterceptors[iLi:], this.Apps[iApp].LocalInterceptors[iLi+1:])
					this.Apps[iApp].LocalInterceptors[len(this.Apps[iApp].LocalInterceptors)-1] = nil
					this.Apps[iApp].LocalInterceptors = this.Apps[iApp].LocalInterceptors[:len(this.Apps[iApp].LocalInterceptors)-1]
					//					this.Apps[iApp].LocalInterceptors = append(this.Apps[iApp].LocalInterceptors[:iLi], this.Apps[iApp].LocalInterceptors[iLi+1:]...)
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Local interceptor not found: " + id)
}
func (this *MasterData) UpdateLI(li *LocalInterceptor) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == li.AppId {
			for iLi, vLi := range this.Apps[iApp].LocalInterceptors {
				if vLi.Id == li.Id && vLi.AppId == li.AppId {
					if li.Name != "__not_set__" {
						vLi.Name = li.Name
					}
					if li.Target != "__not_set__" {
						vLi.Target = li.Target
					}
					if li.Callback != "__not_set__" {
						vLi.Callback = li.Callback
					}
					if li.Type != "__not_set__" {
						vLi.Type = li.Type
					}
					if li.Note != "__not_set__" {
						vLi.Note = li.Note
					}
					this.Apps[iApp].LocalInterceptors[iLi] = vLi
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Local interceptor not found: " + li.Name)
}

func (this *MasterData) AddRI(ri *RemoteInterceptor) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == ri.AppId {
			for _, vRi := range vApp.RemoteInterceptors {
				if vRi.Name == ri.Name && vRi.AppId == ri.AppId {
					return errors.New("Remote interceptor existed: " + ri.Name)
				}
			}
			this.Apps[iApp].RemoteInterceptors = append(this.Apps[iApp].RemoteInterceptors, ri)
			this.Version++
			return masterData.Propagate()
		}
	}
	return errors.New("App does not exist: " + ri.AppId)
}
func (this *MasterData) RemoveRI(id string, appId string) error {
	for iApp, _ := range this.Apps {
		if this.Apps[iApp].Id == appId {
			for iRi, vRi := range this.Apps[iApp].RemoteInterceptors {
				if vRi.Id == id && vRi.AppId == appId {
					copy(this.Apps[iApp].RemoteInterceptors[iRi:], this.Apps[iApp].RemoteInterceptors[iRi+1:])
					this.Apps[iApp].RemoteInterceptors[len(this.Apps[iApp].RemoteInterceptors)-1] = nil
					this.Apps[iApp].RemoteInterceptors = this.Apps[iApp].RemoteInterceptors[:len(this.Apps[iApp].RemoteInterceptors)-1]
					//					this.Apps[iApp].RemoteInterceptors = append(this.Apps[iApp].RemoteInterceptors[:iRi], this.Apps[iApp].RemoteInterceptors[iRi+1:]...)
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Local interceptor not found: " + id)
}
func (this *MasterData) UpdateRI(ri *RemoteInterceptor) error {
	for iApp, vApp := range this.Apps {
		if vApp.Id == ri.AppId {
			for iRi, vRi := range this.Apps[iApp].RemoteInterceptors {
				if vRi.Id == ri.Id && vRi.AppId == ri.AppId {
					if ri.Name != "__not_set__" {
						vRi.Name = ri.Name
					}
					if ri.Target != "__not_set__" {
						vRi.Target = ri.Target
					}
					if ri.Method != "__not_set__" {
						vRi.Method = ri.Method
					}
					if ri.Url != "__not_set__" {
						vRi.Url = ri.Url
					}
					if ri.ActionType != "__not_set__" {
						vRi.ActionType = ri.ActionType
					}
					if ri.Callback != "__not_set__" {
						vRi.Callback = ri.Callback
					}
					if ri.Type != "__not_set__" {
						vRi.Type = ri.Type
					}
					if ri.Note != "__not_set__" {
						vRi.Note = ri.Note
					}
					this.Apps[iApp].RemoteInterceptors[iRi] = vRi
					this.Version++
					return masterData.Propagate()
				}
			}
		}
	}
	return errors.New("Local interceptor not found: " + ri.Name)
}

func AddApiNode(apiNode *ApiNode) error {
	for _, v := range apiNodes {
		if v.Name == apiNode.Name {
			return errors.New("API node existed: " + apiNode.Name)
		}
	}
	apiNodes = append(apiNodes, apiNode)
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
	copy(apiNodes[index:], apiNodes[index+1:])
	apiNodes[len(apiNodes)-1] = nil
	apiNodes = apiNodes[:len(apiNodes)-1]
	//	apiNodes = append(apiNodes[:index], apiNodes[index+1:]...)
	return nil
}

func (this *Query) Reload() error {
	var app *App = nil
	for iApp, vApp := range masterData.Apps {
		if this.AppId == vApp.Id {
			app = masterData.Apps[iApp]
			break
		}
	}

	if app == nil {
		return errors.New("App not found: " + this.AppId)
	}
	if strings.TrimSpace(this.ScriptPath) == "" {
		qFileFound := false
		qFileName := ".netdata/" + app.Name + "/" + this.Name
		if _, err := os.Stat(homeDir + "/" + qFileName); !os.IsNotExist(err) {
			qFileName = homeDir + "/" + qFileName
			qFileFound = true
		}
		if _, err := os.Stat(pwd + "/" + qFileName); !os.IsNotExist(err) {
			qFileName = pwd + "/" + qFileName
			qFileFound = true
		}

		if !qFileFound {
			qFileName += ".sql"
			if _, err := os.Stat(homeDir + "/" + qFileName); !os.IsNotExist(err) {
				qFileName = homeDir + "/" + qFileName
				qFileFound = true
			}
			if _, err := os.Stat(pwd + "/" + qFileName); !os.IsNotExist(err) {
				qFileName = pwd + "/" + qFileName
				qFileFound = true
			}
		}

		content, err := ioutil.ReadFile(qFileName)
		if err != nil {
			return errors.New("Failed to open query file: " + qFileName)
		}
		this.ScriptPath = qFileName
		this.ScriptText = string(content)
	} else {
		content, err := ioutil.ReadFile(this.ScriptPath)
		if err != nil {
			return errors.New("File not found: " + this.ScriptPath)
		}
		this.ScriptText = string(content)
	}
	return nil
}

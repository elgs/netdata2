// master_data
package main

type WsCommand struct {
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
	Port     int64
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

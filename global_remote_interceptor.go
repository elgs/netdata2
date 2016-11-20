// global_remote_interceptor
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/elgs/gorest2"
)

func init() {
	gorest2.RegisterGlobalDataInterceptor(20, &GlobalRemoteInterceptor{Id: "GlobalRemoteInterceptor"})
}

type GlobalRemoteInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func (this *GlobalRemoteInterceptor) executeRemoteInterceptor(tx *sql.Tx, db *sql.DB, context map[string]interface{}, data string, appId string, resourceId string, ri *RemoteInterceptor) error {
	res, status, err := httpRequest(ri.Url, ri.Method, data, -1)
	if err != nil {
		return err
	}
	if status != 200 {
		return errors.New("Client rejected.")
	}
	clientData := string(res)

	sqlScript, err := getQueryText(appId, ri.Callback)
	if err != nil {
		return err
	}
	scripts := sqlScript
	replaceContext := buildReplaceContext(context)

	queryParams, params, err := buildParams(clientData)
	//		fmt.Println(queryParams, params)
	if err != nil {
		return err
	}
	_, err = batchExecuteTx(tx, db, &scripts, queryParams, params, false, replaceContext)
	if err != nil {
		return err
	}
	return nil
}

func (this *GlobalRemoteInterceptor) commonBefore(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, data interface{}) (bool, error) {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	app := context["app"].(*App)
	for _, ri := range app.RemoteInterceptors {
		if ri.Type == "before" && ri.ActionType == action && ri.Target == resourceId && ri.AppId == app.Id {
			payload, err := this.createPayload(resourceId, "before_"+action, data)
			if err != nil {
				return false, err
			}
			err = this.executeRemoteInterceptor(tx, db, context, payload, app.Id, resourceId, ri)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (this *GlobalRemoteInterceptor) commonAfter(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, data interface{}) error {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	app := context["app"].(*App)
	for _, ri := range app.RemoteInterceptors {
		if ri.Type == "after" && ri.ActionType == action && ri.Target == resourceId && ri.AppId == app.Id {
			payload, err := this.createPayload(resourceId, "after_"+action, data)
			if err != nil {
				return err
			}
			err = this.executeRemoteInterceptor(tx, db, context, payload, app.Id, resourceId, ri)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *GlobalRemoteInterceptor) createPayload(target string, action string, data interface{}) (string, error) {
	rts := strings.Split(strings.Replace(target, "`", "", -1), ".")
	target = rts[len(rts)-1]
	m := map[string]interface{}{
		"target": target,
		"action": action,
		"data":   data,
	}
	jsonData, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func (this *GlobalRemoteInterceptor) BeforeExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "exec", map[string]interface{}{"params": *params})
}
func (this *GlobalRemoteInterceptor) AfterExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}, data *[][]interface{}) error {
	return this.commonAfter(nil, db, resourceId, context, "exec", *data)
}

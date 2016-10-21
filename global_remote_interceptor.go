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
	_, err = batchExecuteTx(tx, db, &scripts, queryParams, params, replaceContext)
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
			err = this.executeRemoteInterceptor(tx, db, context, payload, app.Id, resourceId, &ri)
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
			err = this.executeRemoteInterceptor(tx, db, context, payload, app.Id, resourceId, &ri)
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

//func (this *GlobalRemoteInterceptor) BeforeCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "create", data)
//}
//func (this *GlobalRemoteInterceptor) AfterCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
//	return this.commonAfter(nil, db, resourceId, context, "create", data)
//}
//func (this *GlobalRemoteInterceptor) BeforeLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, id string) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "load", map[string]string{"id": id})
//}
//func (this *GlobalRemoteInterceptor) AfterLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data map[string]string) error {
//	return this.commonAfter(nil, db, resourceId, context, "load", data)
//}
//func (this *GlobalRemoteInterceptor) BeforeUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "update", data)
//}
//func (this *GlobalRemoteInterceptor) AfterUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
//	return this.commonAfter(nil, db, resourceId, context, "update", data)
//}
//func (this *GlobalRemoteInterceptor) BeforeDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "duplicate", map[string][]string{"id": id})
//}
//func (this *GlobalRemoteInterceptor) AfterDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string, newId []string) error {
//	return this.commonAfter(nil, db, resourceId, context, "duplicate", map[string][]string{"new_id": newId})
//}
//func (this *GlobalRemoteInterceptor) BeforeDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "delete", map[string][]string{"id": id})
//}
//func (this *GlobalRemoteInterceptor) AfterDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) error {
//	return this.commonAfter(nil, db, resourceId, context, "delete", map[string][]string{"id": id})
//}
//func (this *GlobalRemoteInterceptor) BeforeListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "list_map", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
//}
//func (this *GlobalRemoteInterceptor) AfterListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data *[]map[string]string, total int64) error {
//	return this.commonAfter(nil, db, resourceId, context, "list_map", *data)
//}
//func (this *GlobalRemoteInterceptor) BeforeListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "list_array", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
//}
//func (this *GlobalRemoteInterceptor) AfterListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, headers *[]string, data *[][]string, total int64) error {
//	return this.commonAfter(nil, db, resourceId, context, "list_array", map[string]interface{}{"headers": *headers, "data": *data})
//}
func (this *GlobalRemoteInterceptor) BeforeQueryMap(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_map", map[string]interface{}{"params": *params})
}
func (this *GlobalRemoteInterceptor) AfterQueryMap(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}, data *[]map[string]string) error {
	return this.commonAfter(nil, db, resourceId, context, "query_map", *data)
}
func (this *GlobalRemoteInterceptor) BeforeQueryArray(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_array", map[string]interface{}{"params": *params})
}
func (this *GlobalRemoteInterceptor) AfterQueryArray(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}, headers *[]string, data *[][]string) error {
	return this.commonAfter(nil, db, resourceId, context, "query_array", map[string]interface{}{"headers": *headers, "data": *data})
}
func (this *GlobalRemoteInterceptor) BeforeExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}) (bool, error) {
	return this.commonBefore(tx, nil, resourceId, context, "exec", map[string]interface{}{"params": *params, "query_params": queryParams})
}
func (this *GlobalRemoteInterceptor) AfterExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}, rowsAffectedArray [][]int64) error {
	return this.commonAfter(tx, nil, resourceId, context, "exec", map[string]interface{}{"params": *params, "query_params": queryParams, "rows_affected": rowsAffectedArray})
}

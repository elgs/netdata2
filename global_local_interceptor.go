// global_remote_interceptor
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/elgs/gorest2"
	"github.com/elgs/gosqljson"
	"github.com/elgs/jsonql"
	"github.com/satori/go.uuid"
)

func init() {
	gorest2.RegisterGlobalDataInterceptor(10, &GlobalLocalInterceptor{Id: "GlobalLocalInterceptor"})
}

type GlobalLocalInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func (this *GlobalLocalInterceptor) checkAgainstBeforeLocalInterceptor(tx *sql.Tx, db *sql.DB, context map[string]interface{}, data string, appId string, resourceId string, action string, ri *RemoteInterceptor) (bool, error) {
	res, status, err := httpRequest(ri.Url, ri.Method, data, -1)
	if err != nil {
		return false, err
	}
	if status != 200 {
		return false, errors.New("Client rejected.")
	}
	callback := ri.Callback
	clientData := string(res)

	if strings.TrimSpace(callback) != "" {
		// return a array of array as parameters for callback
		query, err := loadQuery(appId, callback)
		if err != nil {
			return false, err
		}
		scripts := query.Script
		replaceContext := buildReplaceContext(context)
		queryParams, params, err := buildParams(clientData)
		//		fmt.Println(queryParams, params)
		if err != nil {
			return false, err
		}
		_, err = batchExecuteTx(tx, db, &scripts, queryParams, params, replaceContext)
		if err != nil {
			return false, err
		}
	}
	return true, nil

}

func (this *GlobalLocalInterceptor) executeAfterLocalInterceptor(data string, appId string, resourceId string, action string, ri *RemoteInterceptor, context map[string]interface{}) error {
	dataId := strings.Replace(uuid.NewV4().String(), "-", "", -1)
	insert := `INSERT INTO push_notification(ID,PROJECT_ID,TARGET,METHOD,URL,TYPE,ACTION_TYPE,STATUS,DATA,CALLBACK,
	CREATOR_ID,CREATOR_CODE,CREATE_TIME,UPDATER_ID,UPDATER_CODE,UPDATE_TIME) 
	VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	now := time.Now().UTC()
	userId := context["user_id"]
	userCode := context["email"]

	params := []interface{}{dataId, appId, resourceId, ri.Method, ri.Url, "after", action, "0", data, ri.Callback,
		userId, userCode, now, userId, userCode, now}
	defaultDbo, err := gorest2.GetDbo("default")
	if err != nil {
		return err
	}
	defaultDb, err := defaultDbo.GetConn()
	if err != nil {
		return err
	}
	_, err = gosqljson.ExecDb(defaultDb, insert, params...)
	return err
}

func (this *GlobalLocalInterceptor) commonBefore(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, data interface{}) (bool, error) {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	appId := context["app_id"].(string)
	ri := &RemoteInterceptor{}

	criteria := ri.Criteria
	if len(strings.TrimSpace(criteria)) > 0 {
		parser := jsonql.NewQuery(data)
		criteriaResult, err := parser.Query(criteria)
		if err != nil {
			return true, err
		}

		switch v := criteriaResult.(type) {
		case []interface{}:
			if len(v) == 0 {
				return true, nil
			}
		case map[string]interface{}:
			if v == nil {
				return true, nil
			}
		default:
			return true, nil
		}
		data = criteriaResult
	}

	payload, err := this.createPayload(resourceId, "before_"+action, data)
	if err != nil {
		return false, err
	}

	return this.checkAgainstBeforeLocalInterceptor(tx, db, context, payload, appId, resourceId, action, ri)
}

func (this *GlobalLocalInterceptor) commonAfter(resourceId string, context map[string]interface{}, action string, data interface{}) error {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	appId := context["app_id"].(string)
	ri := &RemoteInterceptor{}

	criteria := ri.Criteria
	if len(strings.TrimSpace(criteria)) > 0 {
		parser := jsonql.NewQuery(data)
		criteriaResult, err := parser.Query(criteria)
		if err != nil {
			return err
		}

		switch v := criteriaResult.(type) {
		case []interface{}:
			if len(v) == 0 {
				return nil
			}
		case map[string]interface{}:
			if v == nil {
				return nil
			}
		default:
			return nil
		}
		data = criteriaResult
	}
	payload, err := this.createPayload(resourceId, "after_"+action, data)
	if err != nil {
		return err
	}
	return this.executeAfterLocalInterceptor(payload, appId, resourceId, action, ri, context)
}

func (this *GlobalLocalInterceptor) createPayload(target string, action string, data interface{}) (string, error) {
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

func (this *GlobalLocalInterceptor) BeforeCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	ret, err := this.commonBefore(nil, db, resourceId, context, "create", data)
	if !ret || err != nil {
		return ret, err
	}
	return ret, nil
}
func (this *GlobalLocalInterceptor) AfterCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	err := this.commonAfter(resourceId, context, "create", data)
	if err != nil {
		return err
	}
	return nil
}
func (this *GlobalLocalInterceptor) BeforeLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, id string) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "load", map[string]string{"id": id})
}
func (this *GlobalLocalInterceptor) AfterLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data map[string]string) error {
	return this.commonAfter(resourceId, context, "load", data)
}
func (this *GlobalLocalInterceptor) BeforeUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	ret, err := this.commonBefore(nil, db, resourceId, context, "update", data)
	if !ret || err != nil {
		return ret, err
	}
	return ret, nil
}
func (this *GlobalLocalInterceptor) AfterUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	err := this.commonAfter(resourceId, context, "update", data)
	if err != nil {
		return err
	}
	return nil
}
func (this *GlobalLocalInterceptor) BeforeDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	ret, err := this.commonBefore(nil, db, resourceId, context, "duplicate", map[string][]string{"id": id})
	if !ret || err != nil {
		return ret, err
	}
	return ret, nil
}
func (this *GlobalLocalInterceptor) AfterDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string, newId []string) error {
	err := this.commonAfter(resourceId, context, "duplicate", map[string][]string{"new_id": newId})
	if err != nil {
		return err
	}
	return nil
}
func (this *GlobalLocalInterceptor) BeforeDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	ret, err := this.commonBefore(nil, db, resourceId, context, "delete", map[string][]string{"id": id})
	if !ret || err != nil {
		return ret, err
	}
	return ret, nil
}
func (this *GlobalLocalInterceptor) AfterDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) error {
	err := this.commonAfter(resourceId, context, "delete", map[string][]string{"id": id})
	if err != nil {
		return err
	}
	return nil
}
func (this *GlobalLocalInterceptor) BeforeListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "list_map", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
}
func (this *GlobalLocalInterceptor) AfterListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data *[]map[string]string, total int64) error {
	return this.commonAfter(resourceId, context, "list_map", *data)
}
func (this *GlobalLocalInterceptor) BeforeListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "list_array", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
}
func (this *GlobalLocalInterceptor) AfterListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, headers *[]string, data *[][]string, total int64) error {
	return this.commonAfter(resourceId, context, "list_array", map[string]interface{}{"headers": *headers, "data": *data})
}
func (this *GlobalLocalInterceptor) BeforeQueryMap(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_map", map[string]interface{}{"params": *params})
}
func (this *GlobalLocalInterceptor) AfterQueryMap(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}, data *[]map[string]string) error {
	return this.commonAfter(resourceId, context, "query_map", *data)
}
func (this *GlobalLocalInterceptor) BeforeQueryArray(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_array", map[string]interface{}{"params": *params})
}
func (this *GlobalLocalInterceptor) AfterQueryArray(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}, headers *[]string, data *[][]string) error {
	return this.commonAfter(resourceId, context, "query_array", map[string]interface{}{"headers": *headers, "data": *data})
}
func (this *GlobalLocalInterceptor) BeforeExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}) (bool, error) {
	ret, err := this.commonBefore(tx, nil, resourceId, context, "exec", map[string]interface{}{"params": *params, "query_params": queryParams})
	if !ret || err != nil {
		return ret, err
	}
	return true, nil
}
func (this *GlobalLocalInterceptor) AfterExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}, rowsAffectedArray [][]int64) error {
	err := this.commonAfter(resourceId, context, "exec", map[string]interface{}{"params": *params, "query_params": queryParams, "rows_affected": rowsAffectedArray})
	if err != nil {
		return err
	}
	return nil
}

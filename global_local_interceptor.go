// global_local_interceptor
package main

import (
	"database/sql"
	"strings"

	"github.com/elgs/gorest2"
)

func init() {
	gorest2.RegisterGlobalDataInterceptor(10, &GlobalLocalInterceptor{Id: "GlobalLocalInterceptor"})
}

type GlobalLocalInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func (this *GlobalLocalInterceptor) executeLocalInterceptor(tx *sql.Tx, db *sql.DB, context map[string]interface{}, queryParams []string, data [][]interface{}, appId string, resourceId string, li *LocalInterceptor) error {
	sqlScript, err := getQueryText(appId, li.Callback)
	if err != nil {
		return err
	}
	scripts := sqlScript
	replaceContext := buildReplaceContext(context)

	_, err = batchExecuteTx(tx, db, &scripts, queryParams, data, replaceContext)
	if err != nil {
		return err
	}
	return nil
}

func (this *GlobalLocalInterceptor) commonBefore(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, queryParams []string, data [][]interface{}) (bool, error) {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	app := context["app"].(*App)
	for _, li := range app.LocalInterceptors {
		if li.Type == "before" && li.Target == resourceId && li.AppId == app.Id {
			err := this.executeLocalInterceptor(tx, db, context, queryParams, data, app.Id, resourceId, li)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (this *GlobalLocalInterceptor) commonAfter(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, queryParams []string, data [][]interface{}) error {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	app := context["app"].(*App)
	for _, li := range app.LocalInterceptors {
		if li.Type == "after" && li.Target == resourceId && li.AppId == app.Id {
			err := this.executeLocalInterceptor(tx, db, context, queryParams, data, app.Id, resourceId, li)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//func (this *GlobalLocalInterceptor) BeforeCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "create", data)
//}
//func (this *GlobalLocalInterceptor) AfterCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
//	return this.commonAfter(nil, db, resourceId, context, "create", data)
//}
//func (this *GlobalLocalInterceptor) BeforeLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, id string) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "load", map[string]string{"id": id})
//}
//func (this *GlobalLocalInterceptor) AfterLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data map[string]string) error {
//	return this.commonAfter(nil, db, resourceId, context, "load", data)
//}
//func (this *GlobalLocalInterceptor) BeforeUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "update", data)
//}
//func (this *GlobalLocalInterceptor) AfterUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
//	return this.commonAfter(nil, db, resourceId, context, "update", data)
//}
//func (this *GlobalLocalInterceptor) BeforeDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "duplicate", map[string][]string{"id": id})
//}
//func (this *GlobalLocalInterceptor) AfterDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string, newId []string) error {
//	return this.commonAfter(nil, db, resourceId, context, "duplicate", map[string][]string{"new_id": newId})
//}
//func (this *GlobalLocalInterceptor) BeforeDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "delete", map[string][]string{"id": id})
//}
//func (this *GlobalLocalInterceptor) AfterDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) error {
//	return this.commonAfter(nil, db, resourceId, context, "delete", map[string][]string{"id": id})
//}
//func (this *GlobalLocalInterceptor) BeforeListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "list_map", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
//}
//func (this *GlobalLocalInterceptor) AfterListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data *[]map[string]string, total int64) error {
//	return this.commonAfter(nil, db, resourceId, context, "list_map", *data)
//}
//func (this *GlobalLocalInterceptor) BeforeListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
//	return this.commonBefore(nil, db, resourceId, context, "list_array", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
//}
//func (this *GlobalLocalInterceptor) AfterListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, headers *[]string, data *[][]string, total int64) error {
//	return this.commonAfter(nil, db, resourceId, context, "list_array", map[string]interface{}{"headers": *headers, "data": *data})
//}
func (this *GlobalLocalInterceptor) BeforeQueryMap(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_map", queryParams, [][]interface{}{*params})
}
func (this *GlobalLocalInterceptor) AfterQueryMap(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}, data *[]map[string]string) error {
	return this.commonAfter(nil, db, resourceId, context, "query_map", queryParams, [][]interface{}{*params})
}
func (this *GlobalLocalInterceptor) BeforeQueryArray(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_array", queryParams, [][]interface{}{*params})
}
func (this *GlobalLocalInterceptor) AfterQueryArray(resourceId string, script string, params *[]interface{}, queryParams []string, db *sql.DB, context map[string]interface{}, headers *[]string, data *[][]string) error {
	results := make([][]interface{}, len(*data))
	for i, v := range *data {
		result := make([]interface{}, len(v))
		for ii, vv := range v {
			result[ii] = vv
		}
		results[i] = result
	}
	return this.commonAfter(nil, db, resourceId, context, "query_array", queryParams, results)
}
func (this *GlobalLocalInterceptor) BeforeExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}) (bool, error) {
	return this.commonBefore(tx, nil, resourceId, context, "exec", queryParams, *params)
}
func (this *GlobalLocalInterceptor) AfterExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}, rowsAffectedArray [][]int64) error {
	return this.commonAfter(tx, nil, resourceId, context, "exec", queryParams, *params)
}

// global_local_interceptor
package main

import (
	"database/sql"
	"strings"

	"github.com/elgs/gorest2"
	"github.com/elgs/jsonql"
)

func init() {
	gorest2.RegisterGlobalDataInterceptor(10, &GlobalLocalInterceptor{Id: "GlobalLocalInterceptor"})
}

type GlobalLocalInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func (this *GlobalLocalInterceptor) executeLocalInterceptor(tx *sql.Tx, db *sql.DB, context map[string]interface{}, data interface{}, appId string, resourceId string, li *LocalInterceptor) error {
	query, err := loadQuery(appId, li.Callback)
	if err != nil {
		return err
	}
	scripts := query.Script
	replaceContext := buildReplaceContext(context)

	_, err = batchExecuteTx(tx, db, &scripts, []string{}, data.([][]interface{}), replaceContext)
	if err != nil {
		return err
	}
	return nil
}

func (this *GlobalLocalInterceptor) commonBefore(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, data interface{}) (bool, error) {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	app := context["app"].(*App)
	for _, li := range app.LocalInterceptors {
		if li.Type == "before" && li.Target == resourceId && li.AppId == app.Id {
			if len(strings.TrimSpace(li.Criteria)) > 0 {
				parser := jsonql.NewQuery(data)
				criteriaResult, err := parser.Query(li.Criteria)
				if err != nil {
					return true, err
				}
				err = this.executeLocalInterceptor(tx, db, context, criteriaResult, app.Id, resourceId, &li)
				if err != nil {
					return false, err
				}
			} else {
				err := this.executeLocalInterceptor(tx, db, context, data, app.Id, resourceId, &li)
				if err != nil {
					return false, err
				}
			}
		}
	}
	return true, nil
}

func (this *GlobalLocalInterceptor) commonAfter(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, data interface{}) error {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	app := context["app"].(*App)
	for _, li := range app.LocalInterceptors {
		if li.Type == "after" && li.Target == resourceId && li.AppId == app.Id {
			if len(strings.TrimSpace(li.Criteria)) > 0 {
				parser := jsonql.NewQuery(data)
				criteriaResult, err := parser.Query(li.Criteria)
				if err != nil {
					return err
				}
				err = this.executeLocalInterceptor(tx, db, context, criteriaResult, app.Id, resourceId, &li)
				if err != nil {
					return err
				}
			} else {
				err := this.executeLocalInterceptor(tx, db, context, data, app.Id, resourceId, &li)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (this *GlobalLocalInterceptor) BeforeCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "create", data)
}
func (this *GlobalLocalInterceptor) AfterCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	return this.commonAfter(nil, db, resourceId, context, "create", data)
}
func (this *GlobalLocalInterceptor) BeforeLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, id string) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "load", map[string]string{"id": id})
}
func (this *GlobalLocalInterceptor) AfterLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data map[string]string) error {
	return this.commonAfter(nil, db, resourceId, context, "load", data)
}
func (this *GlobalLocalInterceptor) BeforeUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "update", data)
}
func (this *GlobalLocalInterceptor) AfterUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	return this.commonAfter(nil, db, resourceId, context, "update", data)
}
func (this *GlobalLocalInterceptor) BeforeDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "duplicate", map[string][]string{"id": id})
}
func (this *GlobalLocalInterceptor) AfterDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string, newId []string) error {
	return this.commonAfter(nil, db, resourceId, context, "duplicate", map[string][]string{"new_id": newId})
}
func (this *GlobalLocalInterceptor) BeforeDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "delete", map[string][]string{"id": id})
}
func (this *GlobalLocalInterceptor) AfterDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) error {
	return this.commonAfter(nil, db, resourceId, context, "delete", map[string][]string{"id": id})
}
func (this *GlobalLocalInterceptor) BeforeListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "list_map", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
}
func (this *GlobalLocalInterceptor) AfterListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data *[]map[string]string, total int64) error {
	return this.commonAfter(nil, db, resourceId, context, "list_map", *data)
}
func (this *GlobalLocalInterceptor) BeforeListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "list_array", map[string]interface{}{"fields": fields, "filter": *filter, "sort": *sort, "group": *group, "start": start, "limit": limit})
}
func (this *GlobalLocalInterceptor) AfterListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, headers *[]string, data *[][]string, total int64) error {
	return this.commonAfter(nil, db, resourceId, context, "list_array", map[string]interface{}{"headers": *headers, "data": *data})
}
func (this *GlobalLocalInterceptor) BeforeQueryMap(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_map", map[string]interface{}{"params": *params})
}
func (this *GlobalLocalInterceptor) AfterQueryMap(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}, data *[]map[string]string) error {
	return this.commonAfter(nil, db, resourceId, context, "query_map", *data)
}
func (this *GlobalLocalInterceptor) BeforeQueryArray(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}) (bool, error) {
	return this.commonBefore(nil, db, resourceId, context, "query_array", map[string]interface{}{"params": *params})
}
func (this *GlobalLocalInterceptor) AfterQueryArray(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}, headers *[]string, data *[][]string) error {
	return this.commonAfter(nil, db, resourceId, context, "query_array", map[string]interface{}{"headers": *headers, "data": *data})
}
func (this *GlobalLocalInterceptor) BeforeExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}) (bool, error) {
	return this.commonBefore(tx, nil, resourceId, context, "exec", map[string]interface{}{"params": *params, "query_params": queryParams})
}
func (this *GlobalLocalInterceptor) AfterExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}, rowsAffectedArray [][]int64) error {
	return this.commonAfter(tx, nil, resourceId, context, "exec", map[string]interface{}{"params": *params, "query_params": queryParams, "rows_affected": rowsAffectedArray})
}

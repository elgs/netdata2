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

func (this *GlobalLocalInterceptor) executeLocalInterceptor(tx *sql.Tx, db *sql.DB, context map[string]interface{}, queryParams map[string]string, data [][]interface{}, appId string, resourceId string, li *LocalInterceptor) error {
	sqlScript, err := getQueryText(appId, li.Callback)
	if err != nil {
		return err
	}
	scripts := sqlScript
	replaceContext := buildReplaceContext(context)

	_, err = batchExecuteTx(tx, db, &scripts, queryParams, data, false, "", replaceContext)
	if err != nil {
		return err
	}
	return nil
}

func (this *GlobalLocalInterceptor) commonBefore(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, queryParams map[string]string, data [][]interface{}) error {
	rts := strings.Split(strings.Replace(resourceId, "`", "", -1), ".")
	resourceId = rts[len(rts)-1]
	app := context["app"].(*App)
	for _, li := range app.LocalInterceptors {
		if li.Type == "before" && li.Target == resourceId && li.AppId == app.Id {
			err := this.executeLocalInterceptor(tx, db, context, queryParams, data, app.Id, resourceId, li)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *GlobalLocalInterceptor) commonAfter(tx *sql.Tx, db *sql.DB, resourceId string, context map[string]interface{}, action string, queryParams map[string]string, data [][]interface{}) error {
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

func (this *GlobalLocalInterceptor) BeforeExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}) error {
	return this.commonBefore(nil, db, resourceId, context, "query_map", queryParams, *params)
}
func (this *GlobalLocalInterceptor) AfterExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}, data *[][]interface{}) error {
	return this.commonAfter(nil, db, resourceId, context, "query_map", queryParams, *params)
}

// nd_data_operator
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/elgs/gorest2"
	"github.com/elgs/gosplitargs"
	"github.com/elgs/gosqljson"
)

type NdDataOperator struct {
	*gorest2.MySqlDataOperator
}

func NewDbo(ds, dbType string) gorest2.DataOperator {
	return &NdDataOperator{
		MySqlDataOperator: &gorest2.MySqlDataOperator{
			Ds:     ds,
			DbType: dbType,
		},
		//		QueryRegistry: make(map[string]map[string]string),
	}
}

func loadQuery(projectId, queryName string) (string, error) {
	var app *App = nil
	for _, vApp := range masterData.Apps {
		if projectId == vApp.Id {
			app = &vApp
			break
		}
	}

	if app == nil {
		return "", errors.New("App not found: " + projectId)
	}

	for iQuery, vQuery := range app.Queries {
		if vQuery.Name == queryName {
			q := app.Queries[iQuery]
			if strings.HasPrefix(q.Script, "./") || strings.HasPrefix(q.Script, "/") {
				content, err := ioutil.ReadFile(q.Script)
				if err != nil {
					return "", errors.New("File not found: " + q.Script)
				}
				return string(content), nil
			} else if strings.TrimSpace(q.Script) == "" {
				qFileName := ".netdata/" + app.Name + "/" + q.Name
				if _, err := os.Stat(homeDir + "/" + qFileName); os.IsExist(err) {
					qFileName = homeDir + "/" + qFileName
				}
				if _, err := os.Stat(pwd + "/" + qFileName); os.IsExist(err) {
					qFileName = pwd + "/" + qFileName
				}

				content, err := ioutil.ReadFile(qFileName)
				if err != nil {
					return "", errors.New("Failed to open query file not found: " + qFileName)
				}
				return string(content), nil
			} else {
				return q.Script, nil
			}
		}
	}
	return "", errors.New("Query not found: " + queryName)
}

func (this *NdDataOperator) QueryMap(tableId string, params []interface{}, queryParams []string, context map[string]interface{}) ([]map[string]string, error) {
	projectId := context["app_id"].(string)
	sqlScript, err := loadQuery(projectId, tableId)
	if err != nil {
		return nil, err
	}

	ret := make([]map[string]string, 0)

	script := sqlScript

	count, err := gosplitargs.CountSeparators(script, "\\?")
	if err != nil {
		return ret, err
	}
	if count > len(params) {
		return nil, errors.New(fmt.Sprintln("Incorrect param count. Expected: ", count, " actual: ", len(params)))
	}

	for i, v := range queryParams {
		script = strings.Replace(script, fmt.Sprint("$", i), v, -1)
	}

	db, err := this.GetConn()
	if err != nil {
		return ret, err
	}

	globalDataInterceptors, globalSortedKeys := gorest2.GetGlobalDataInterceptors()
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		ctn, err := globalDataInterceptor.BeforeQueryMap(tableId, script, &params, queryParams, db, context)
		if !ctn {
			return ret, err
		}
	}
	dataInterceptors, sortedKeys := gorest2.GetDataInterceptors(tableId)
	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			ctn, err := dataInterceptor.BeforeQueryMap(tableId, script, &params, queryParams, db, context)
			if !ctn {
				return ret, err
			}
		}
	}

	if clientIp, ok := context["client_ip"].(string); ok {
		script = strings.Replace(script, "__ip__", clientIp, -1)
	}
	if tokenUserId, ok := context["token_user_id"].(string); ok {
		script = strings.Replace(script, "__token_user_id__", tokenUserId, -1)
	}
	if tokenUserCode, ok := context["token_user_code"].(string); ok {
		script = strings.Replace(script, "__token_user_code__", tokenUserCode, -1)
	}
	if loginUserId, ok := context["user_id"].(string); ok {
		script = strings.Replace(script, "__login_user_id__", loginUserId, -1)
	}
	if loginUserCode, ok := context["email"].(string); ok {
		script = strings.Replace(script, "__login_user_code__", loginUserCode, -1)
	}

	c := context["case"].(string)
	m, err := gosqljson.QueryDbToMap(db, c, script, params[:count]...)
	if err != nil {
		fmt.Println(err)
		return ret, err
	}

	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			dataInterceptor.AfterQueryMap(tableId, script, &params, queryParams, db, context, &m)
		}
	}
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		globalDataInterceptor.AfterQueryMap(tableId, script, &params, queryParams, db, context, &m)
	}

	return m, err
}
func (this *NdDataOperator) QueryArray(tableId string, params []interface{}, queryParams []string, context map[string]interface{}) ([]string, [][]string, error) {
	projectId := context["app_id"].(string)
	sqlScript, err := loadQuery(projectId, tableId)
	if err != nil {
		return nil, nil, err
	}

	script := sqlScript
	count, err := gosplitargs.CountSeparators(script, "\\?")
	if err != nil {
		return nil, nil, err
	}
	if count > len(params) {
		return nil, nil, errors.New(fmt.Sprintln("Incorrect param count. Expected: ", count, " actual: ", len(params)))
	}

	for i, v := range queryParams {
		script = strings.Replace(script, fmt.Sprint("$", i), v, -1)
	}

	db, err := this.GetConn()
	if err != nil {
		return nil, nil, err
	}

	globalDataInterceptors, globalSortedKeys := gorest2.GetGlobalDataInterceptors()
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		ctn, err := globalDataInterceptor.BeforeQueryArray(tableId, script, &params, queryParams, db, context)
		if !ctn {
			return nil, nil, err
		}
	}
	dataInterceptors, sortedKeys := gorest2.GetDataInterceptors(tableId)
	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			ctn, err := dataInterceptor.BeforeQueryArray(tableId, script, &params, queryParams, db, context)
			if !ctn {
				return nil, nil, err
			}
		}
	}

	if clientIp, ok := context["client_ip"].(string); ok {
		script = strings.Replace(script, "__ip__", clientIp, -1)
	}
	if tokenUserId, ok := context["token_user_id"].(string); ok {
		script = strings.Replace(script, "__token_user_id__", tokenUserId, -1)
	}
	if tokenUserCode, ok := context["token_user_code"].(string); ok {
		script = strings.Replace(script, "__token_user_code__", tokenUserCode, -1)
	}
	if loginUserId, ok := context["user_id"].(string); ok {
		script = strings.Replace(script, "__login_user_id__", loginUserId, -1)
	}
	if loginUserCode, ok := context["email"].(string); ok {
		script = strings.Replace(script, "__login_user_code__", loginUserCode, -1)
	}

	c := context["case"].(string)
	h, a, err := gosqljson.QueryDbToArray(db, c, script, params[:count]...)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			dataInterceptor.AfterQueryArray(tableId, script, &params, queryParams, db, context, &h, &a)
		}
	}
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		globalDataInterceptor.AfterQueryArray(tableId, script, &params, queryParams, db, context, &h, &a)
	}

	return h, a, err
}
func (this *NdDataOperator) Exec(tableId string, params [][]interface{}, queryParams []string, context map[string]interface{}) ([][]int64, error) {
	projectId := context["app_id"].(string)

	sqlScript, err := loadQuery(projectId, tableId)
	if err != nil {
		return nil, err
	}
	scripts := sqlScript

	db, err := this.GetConn()
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	globalDataInterceptors, globalSortedKeys := gorest2.GetGlobalDataInterceptors()
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		ctn, err := globalDataInterceptor.BeforeExec(tableId, scripts, &params, queryParams, tx, context)
		if !ctn {
			tx.Rollback()
			return nil, err
		}
	}
	dataInterceptors, sortedKeys := gorest2.GetDataInterceptors(tableId)
	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			ctn, err := dataInterceptor.BeforeExec(tableId, scripts, &params, queryParams, tx, context)
			if !ctn {
				tx.Rollback()
				return nil, err
			}
		}
	}

	replaceContext := buildReplaceContext(context)
	rowsAffectedArray, err := batchExecuteTx(tx, nil, &scripts, queryParams, params, replaceContext)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			err := dataInterceptor.AfterExec(tableId, scripts, &params, queryParams, tx, context, rowsAffectedArray)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		err := globalDataInterceptor.AfterExec(tableId, scripts, &params, queryParams, tx, context, rowsAffectedArray)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()

	return rowsAffectedArray, err
}

func MakeGetDbo(dbType string, masterData *MasterData) func(id string) (gorest2.DataOperator, error) {
	return func(id string) (gorest2.DataOperator, error) {
		ret := gorest2.DboRegistry[id]
		if ret != nil {
			return ret, nil
		}

		var app *App = nil
		for _, a := range masterData.Apps {
			if a.Id == id {
				app = &a
				break
			}
		}
		if app == nil {
			return nil, errors.New("App not found: " + id)
		}

		var dn *DataNode = nil
		for _, vDn := range masterData.DataNodes {
			if app.DataNodeId == vDn.Id {
				dn = &vDn
				break
			}
		}

		if dn == nil {
			return nil, errors.New("Data node not found: " + app.DataNodeId)
		}

		ds := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", app.DbName, id, dn.Host, dn.Port, "nd_"+app.DbName)
		ret = NewDbo(ds, dbType)
		gorest2.DboRegistry[id] = ret
		return ret, nil
	}
}

/*
func (this *NdDataOperator) ExecX(tableId string, params [][]interface{}, queryParams []string, context map[string]interface{}) ([]gorest2.NdResult, error) {
	projectId := context["app_id"].(string)

	query, err := loadQuery(projectId, tableId)
	if err != nil {
		return nil, err
	}
	scripts := query["script"]

	db, err := this.GetConn()
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	globalDataInterceptors, globalSortedKeys := gorest2.GetGlobalDataInterceptors()
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		ctn, err := globalDataInterceptor.BeforeExecX(tableId, scripts, &params, queryParams, tx, context)
		if !ctn {
			tx.Rollback()
			return nil, err
		}
	}
	dataInterceptors, sortedKeys := gorest2.GetDataInterceptors(tableId)
	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			ctn, err := dataInterceptor.BeforeExecX(tableId, scripts, &params, queryParams, tx, context)
			if !ctn {
				tx.Rollback()
				return nil, err
			}
		}
	}

	replaceContext := buildReplaceContext(context)
	rowsAffectedArray, err := batchExecuteTx(tx, nil, &scripts, queryParams, params, replaceContext)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			err := dataInterceptor.AfterExecX(tableId, scripts, &params, queryParams, tx, context, rowsAffectedArray)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		err := globalDataInterceptor.AfterExecX(tableId, scripts, &params, queryParams, tx, context, rowsAffectedArray)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()

	return rowsAffectedArray, err
}
*/

// nd_data_operator
package main

import (
	"errors"
	"fmt"
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

func loadQuery(projectId, queryName string) (map[string]string, error) {
	key := fmt.Sprint("query:", projectId, ":", queryName)
	queryMap := gorest2.RedisLocal.HGetAllMap(key).Val()
	if len(queryMap) > 0 {
		return queryMap, nil
	}

	defaultDbo, err := gorest2.GetDbo("default")
	defaultDb, err := defaultDbo.GetConn()
	if err != nil {
		return nil, err
	}
	queryData, err := gosqljson.QueryDbToMap(defaultDb, "upper",
		"SELECT * FROM query WHERE PROJECT_ID=? AND NAME=?", projectId, queryName)
	if err != nil {
		return nil, err
	}
	if len(queryData) == 0 {
		return nil, errors.New("Query not found.")
	}

	err = gorest2.RedisMaster.HMSet(key, "name", queryData[0]["NAME"], "script", queryData[0]["SCRIPT"]).Err()
	return queryData[0], nil
}

func (this *NdDataOperator) QueryMap(tableId string, params []interface{}, queryParams []string, context map[string]interface{}) ([]map[string]string, error) {
	projectId := context["app_id"].(string)
	query, err := loadQuery(projectId, tableId)
	if err != nil {
		return nil, err
	}

	ret := make([]map[string]string, 0)

	script := query["script"]

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
		ctn, err := globalDataInterceptor.BeforeQueryMap(tableId, script, &params, db, context)
		if !ctn {
			return ret, err
		}
	}
	dataInterceptors, sortedKeys := gorest2.GetDataInterceptors(tableId)
	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			ctn, err := dataInterceptor.BeforeQueryMap(tableId, script, &params, db, context)
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
			dataInterceptor.AfterQueryMap(tableId, script, &params, db, context, &m)
		}
	}
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		globalDataInterceptor.AfterQueryMap(tableId, script, &params, db, context, &m)
	}

	return m, err
}
func (this *NdDataOperator) QueryArray(tableId string, params []interface{}, queryParams []string, context map[string]interface{}) ([]string, [][]string, error) {
	projectId := context["app_id"].(string)
	query, err := loadQuery(projectId, tableId)
	if err != nil {
		return nil, nil, err
	}

	script := query["script"]
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
		ctn, err := globalDataInterceptor.BeforeQueryArray(tableId, script, &params, db, context)
		if !ctn {
			return nil, nil, err
		}
	}
	dataInterceptors, sortedKeys := gorest2.GetDataInterceptors(tableId)
	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			ctn, err := dataInterceptor.BeforeQueryArray(tableId, script, &params, db, context)
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
			dataInterceptor.AfterQueryArray(tableId, script, &params, db, context, &h, &a)
		}
	}
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		globalDataInterceptor.AfterQueryArray(tableId, script, &params, db, context, &h, &a)
	}

	return h, a, err
}
func (this *NdDataOperator) Exec(tableId string, params [][]interface{}, queryParams []string, context map[string]interface{}) ([][]int64, error) {
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

		//		query := `SELECT data_store.*,
		//			CONCAT_WS('_','nd',project.PROJECT_KEY) AS DB,project.ID AS PROJECT_ID,project.PROJECT_KEY FROM project
		//			INNER JOIN data_store ON project.DATA_STORE_NAME=data_store.DATA_STORE_NAME WHERE project.ID=?`

		for _, app := range masterData.Apps {
			fmt.Println(app)
		}

		ds := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", "PROJECT_KEY", id, "HOST", "PORT", "DB")
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

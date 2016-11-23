// nd_data_operator
package main

import (
	"errors"
	"fmt"

	"github.com/elgs/gorest2"
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
	}
}

func getQueryText(projectId, queryName string) (string, error) {
	var app *App = nil
	for iApp, vApp := range masterData.Apps {
		if projectId == vApp.Id {
			app = masterData.Apps[iApp]
			break
		}
	}

	if app == nil {
		return "", errors.New("App not found: " + projectId)
	}

	for _, vQuery := range app.Queries {
		if vQuery.Name == queryName {
			return vQuery.ScriptText, nil
		}
	}
	return "", errors.New("Query not found: " + queryName)
}

func (this *NdDataOperator) Exec(tableId string, params [][]interface{}, queryParams map[string]string, array bool, context map[string]interface{}) ([][]interface{}, error) {
	projectId := context["app_id"].(string)
	theCase := context["case"].(string)
	sqlScript, err := getQueryText(projectId, tableId)
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
		ctn, err := globalDataInterceptor.BeforeExec(tableId, scripts, &params, queryParams, array, db, context)
		if !ctn {
			return nil, err
		}
	}
	dataInterceptors, sortedKeys := gorest2.GetDataInterceptors(tableId)
	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			ctn, err := dataInterceptor.BeforeExec(tableId, scripts, &params, queryParams, array, db, context)
			if !ctn {
				return nil, err
			}
		}
	}

	replaceContext := buildReplaceContext(context)
	retArray, err := batchExecuteTx(tx, nil, &scripts, queryParams, params, array, theCase, replaceContext)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, k := range sortedKeys {
		dataInterceptor := dataInterceptors[k]
		if dataInterceptor != nil {
			dataInterceptor.AfterExec(tableId, scripts, &params, queryParams, array, db, context, &retArray)
		}
	}
	for _, k := range globalSortedKeys {
		globalDataInterceptor := globalDataInterceptors[k]
		globalDataInterceptor.AfterExec(tableId, scripts, &params, queryParams, array, db, context, &retArray)
	}

	tx.Commit()

	return retArray, err
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
				app = a
				break
			}
		}
		if app == nil {
			return nil, errors.New("App not found: " + id)
		}

		var dn *DataNode = nil
		for _, vDn := range masterData.DataNodes {
			if app.DataNodeId == vDn.Id {
				dn = vDn
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

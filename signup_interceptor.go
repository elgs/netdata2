// login_interceptor
package main

import (
	"database/sql"

	"github.com/elgs/gorest2"
)

func init() {
	tableId := "signup"
	gorest2.RegisterDataInterceptor(tableId, 0, &SginupInterceptor{Id: tableId})
}

type SginupInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func (this *SginupInterceptor) AfterExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}, data *[][]interface{}) error {
	userInfo := (*data)[0][5]
	if userMap, ok := userInfo.([]map[string]string); ok {
		SendMail("UpRun User Verification", userMap[0]["VERIFICATION_CODE"], userMap[0]["EMAIL"])
	}
	(*data)[0][5] = ""
	return nil
}

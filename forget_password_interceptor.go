// login_interceptor
package main

import (
	"database/sql"

	"github.com/elgs/gorest2"
)

func init() {
	tableId := "forget_password"
	gorest2.RegisterDataInterceptor(tableId, 0, &ForgetPasswordInterceptor{Id: tableId})
}

type ForgetPasswordInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func (this *ForgetPasswordInterceptor) AfterExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}, data *[][]interface{}) error {
	userInfo := (*data)[0][3]
	if userMap, ok := userInfo.([]map[string]string); ok {
		if len(userMap) > 0 {
			SendMail("UpRun User Verification", userMap[0]["VERIFICATION_CODE"], userMap[0]["EMAIL"])
		}
	}
	(*data)[0][3] = ""
	return nil
}

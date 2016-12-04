// login_interceptor
package main

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/elgs/gorest2"
)

func init() {
	tableId := "login"
	gorest2.RegisterDataInterceptor(tableId, 0, &LoginInterceptor{Id: tableId})
}

type LoginInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func (this *LoginInterceptor) AfterExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}, data *[][]interface{}) error {
	// if the query name is login, encrypt the query result into a jwt token.
	tokenData := (*data)[0][0]
	if v, ok := tokenData.([]map[string]string); ok && len(v) > 0 {
		t, err := convertMapOfStringsToMapOfInterfaces(v[0])
		if err != nil {
			return err
		}
		t["exp"] = time.Now().Add(time.Hour * 72).Unix()
		tokenPayload, err := json.Marshal(t)
		if err != nil {
			return err
		}
		s, err := createJwtToken(string(tokenPayload))
		if err != nil {
			return err
		}
		(*data)[0][0] = s
	}
	return nil
}

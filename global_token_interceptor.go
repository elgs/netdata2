package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/elgs/gorest2"
)

func init() {
	gorest2.RegisterGlobalDataInterceptor(0, &GlobalTokenInterceptor{Id: "GlobalTokenInterceptor"})
}

type GlobalTokenInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

func checkAccessPermission(targets, tableId, mode, op string) bool {
	tableMatch, opMatch := false, true
	if targets == "*" {
		tableMatch = true
	} else {
		ts := strings.Split(strings.Replace(tableId, "`", "", -1), ".")
		tableName := ts[len(ts)-1]

		targetsArray := strings.Split(targets, ",")
		for _, target := range targetsArray {
			if target == tableName {
				tableMatch = true
				break
			}
		}
	}
	if !tableMatch {
		return false
	}
	for _, c := range op {
		if !strings.ContainsRune(mode, c) {
			return false
		}
	}
	return tableMatch && opMatch
}

func checkProjectToken(context map[string]interface{}, tableId string, op string) (bool, error) {

	token := context["api_token"].(string)
	if _, ok := context["app"]; !ok {
		appId := context["app_id"].(string)
		for _, a := range masterData.Apps {
			if a.Id == appId {
				context["app"] = a
				break
			}
		}
	}

	app := context["app"].(*App)

	for _, t := range app.Tokens {
		if t.AppId == app.Id && t.Id == token {
			if checkAccessPermission(t.Target, tableId, t.Mode, op) {
				return true, nil
			} else {
				return false, errors.New("Authentication failed.")
			}
			break
		}
	}
	return false, errors.New("Authentication failed.")
}

func (this *GlobalTokenInterceptor) BeforeCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	ctn, err := checkProjectToken(context, resourceId, "w")
	if ctn && err == nil {
		for _, data1 := range data {
			data1["CREATED_AT"] = time.Now().UTC()
			data1["UPDATED_AT"] = time.Now().UTC()
			if userId, found := context["user_id"]; found {
				data1["CREATED_BY"] = userId
				data1["UPDATED_BY"] = userId
			}
		}
	}
	return ctn, err
}
func (this *GlobalTokenInterceptor) AfterCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, id string) (bool, error) {
	return checkProjectToken(context, resourceId, "r")
}
func (this *GlobalTokenInterceptor) AfterLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data map[string]string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	ctn, err := checkProjectToken(context, resourceId, "w")
	if ctn && err == nil {
		for _, data1 := range data {
			data1["UPDATED_AT"] = time.Now().UTC()
			if userId, found := context["user_id"]; found {
				data1["UPDATED_BY"] = userId
			}
		}
	}
	return ctn, err
}
func (this *GlobalTokenInterceptor) AfterUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	return checkProjectToken(context, resourceId, "w")
}
func (this *GlobalTokenInterceptor) AfterDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string, newId []string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	return checkProjectToken(context, resourceId, "w")
}
func (this *GlobalTokenInterceptor) AfterDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	return checkProjectToken(context, resourceId, "r")
}
func (this *GlobalTokenInterceptor) AfterListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data *[]map[string]string, total int64) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	return checkProjectToken(context, resourceId, "r")
}
func (this *GlobalTokenInterceptor) AfterListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, headers *[]string, data *[][]string, total int64) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}) (bool, error) {
	return checkProjectToken(context, resourceId, "rx")
}
func (this *GlobalTokenInterceptor) AfterExec(resourceId string, script string, params *[][]interface{}, queryParams map[string]string, array bool, db *sql.DB, context map[string]interface{}, data *[][]interface{}) error {
	// if the query name is login, encrypt the query result into a jwt token.
	if resourceId == "login" {
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
	}
	return nil
}

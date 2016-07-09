package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elgs/gorest2"
	"github.com/elgs/gosqljson"
)

func init() {
	gorest2.RegisterGlobalDataInterceptor(20, &GlobalTokenInterceptor{Id: "GlobalTokenInterceptor"})
}

type GlobalTokenInterceptor struct {
	*gorest2.DefaultDataInterceptor
	Id string
}

// server, client, server, client
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
	projectId := context["app"].(string)
	token := context["token"].(string)
	key := fmt.Sprint("token:", projectId, ":", token)
	tokenMap := gorest2.RedisLocal.HGetAllMap(key).Val()
	if projectId != "" && token != "" && len(tokenMap) > 0 &&
		len(tokenMap["token_user_id"]) > 0 && len(tokenMap["token_user_code"]) > 0 {
		if checkAccessPermission(tokenMap["targets"], tableId, tokenMap["mode"], op) {
			context["token_user_id"] = tokenMap["token_user_id"]
			context["token_user_code"] = tokenMap["token_user_code"]
			return true, nil
		} else {
			return false, errors.New("Authentication failed.")
		}
	}

	defaultDbo := gorest2.GetDbo("default")
	defaultDb, err := defaultDbo.GetConn()
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	userData, err := gosqljson.QueryDbToMap(defaultDb, "upper",
		"SELECT * FROM token WHERE PROJECT_ID=? AND TOKEN=? AND STATUS=?", projectId, token, "0")
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	if userData != nil && len(userData) == 1 {
		record := userData[0]
		tokenUserId := record["CREATOR_ID"]
		tokenUserCode := record["CREATOR_CODE"]
		context["token_user_id"] = tokenUserId
		context["token_user_code"] = tokenUserCode
		err := gorest2.RedisMaster.HMSet(key, "targets", record["TARGETS"], "mode", record["MODE"],
			"token_user_id", tokenUserId, "token_user_code", tokenUserCode).Err()
		if err != nil {
			return false, err
		}
		if checkAccessPermission(record["TARGETS"], tableId, record["MODE"], op) {
			return true, nil
		} else {
			return false, errors.New("Authentication failed.")
		}
	} else {
		userData, err := gosqljson.QueryDbToMap(defaultDb, "upper",
			`SELECT u.ID,u.EMAIL,u.TOKEN_KEY AS TOKEN,up.PROJECT_ID FROM user AS u INNER JOIN user_project AS up ON u.EMAIL=up.USER_EMAIL 
			WHERE u.TOKEN_KEY=? AND up.PROJECT_ID=? AND u.STATUS=? AND up.STATUS=?`,
			token, projectId, "0", "0")
		if err != nil {
			fmt.Println(err)
			return false, err
		}
		if userData != nil && len(userData) > 0 {
			record := userData[0]
			tokenUserId := record["ID"]
			tokenUserCode := record["EMAIL"]
			context["token_user_id"] = tokenUserId
			context["token_user_code"] = tokenUserCode
			err := gorest2.RedisMaster.HMSet(key, "targets", "*", "mode", "rwx",
				"token_user_id", tokenUserId, "token_user_code", tokenUserCode).Err()
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, errors.New("Authentication failed.")
}

func (this *GlobalTokenInterceptor) BeforeCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	ctn, err := checkProjectToken(context, resourceId, "w")
	if ctn && err == nil {
		if context["meta"] != nil && context["meta"].(bool) {
			for _, data1 := range data {
				data1["CREATE_TIME"] = time.Now().UTC()
				data1["UPDATE_TIME"] = time.Now().UTC()
				if userId, found := context["user_id"]; found {
					data1["CREATOR_ID"] = userId
					data1["UPDATER_ID"] = userId
				}
				if email, found := context["email"]; found {
					data1["CREATOR_CODE"] = email
					data1["UPDATER_CODE"] = email
				}
			}
		}
	}
	return ctn, err
}
func (this *GlobalTokenInterceptor) AfterCreate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, id string) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "r")
}
func (this *GlobalTokenInterceptor) AfterLoad(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data map[string]string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	ctn, err := checkProjectToken(context, resourceId, "w")
	if ctn && err == nil {
		for _, data1 := range data {
			if context["meta"] != nil && context["meta"].(bool) {
				data1["UPDATE_TIME"] = time.Now().UTC()
			}
			if userId, found := context["user_id"]; found {
				data1["UPDATER_ID"] = userId
			}
			if email, found := context["email"]; found {
				data1["UPDATER_CODE"] = email
			}
		}
	}
	return ctn, err
}
func (this *GlobalTokenInterceptor) AfterUpdate(resourceId string, db *sql.DB, context map[string]interface{}, data []map[string]interface{}) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "w")
}
func (this *GlobalTokenInterceptor) AfterDuplicate(resourceId string, db *sql.DB, context map[string]interface{}, id []string, newId []string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "w")
}
func (this *GlobalTokenInterceptor) AfterDelete(resourceId string, db *sql.DB, context map[string]interface{}, id []string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "r")
}
func (this *GlobalTokenInterceptor) AfterListMap(resourceId string, db *sql.DB, fields string, context map[string]interface{}, data *[]map[string]string, total int64) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, filter *string, sort *string, group *string, start int64, limit int64) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "r")
}
func (this *GlobalTokenInterceptor) AfterListArray(resourceId string, db *sql.DB, fields string, context map[string]interface{}, headers *[]string, data *[][]string, total int64) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeQueryMap(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "rx")
}
func (this *GlobalTokenInterceptor) AfterQueryMap(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}, data *[]map[string]string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeQueryArray(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "rx")
}
func (this *GlobalTokenInterceptor) AfterQueryArray(resourceId string, script string, params *[]interface{}, db *sql.DB, context map[string]interface{}, headers *[]string, data *[][]string) error {
	return nil
}
func (this *GlobalTokenInterceptor) BeforeExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}) (bool, error) {
	if isDefaultProjectRequest(context) {
		return true, nil
	}
	return checkProjectToken(context, resourceId, "wx")
}
func (this *GlobalTokenInterceptor) AfterExec(resourceId string, scripts string, params *[][]interface{}, queryParams []string, tx *sql.Tx, context map[string]interface{}, rowsAffectedArray [][]int64) error {
	return nil
}

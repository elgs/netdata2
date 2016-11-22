// ndutils
package main

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/elgs/gojq"
	"github.com/elgs/gosplitargs"
	"github.com/elgs/gosqljson"
)

func httpRequest(url string, method string, data string, maxReadLimit int64) ([]byte, int, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if maxReadLimit >= 0 {
		res.Body = &LimitedReadCloser{res.Body, maxReadLimit}
	}

	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	defer res.Body.Close()
	defer tr.CloseIdleConnections()

	return result, res.StatusCode, err
}

func batchExecuteTx(tx *sql.Tx, db *sql.DB, script *string, scriptParams map[string]string, params [][]interface{}, array bool, theCase string, replaceContext map[string]string) ([][]interface{}, error) {
	ret := [][]interface{}{}

	innerTrans := false
	if tx == nil {
		var err error
		tx, err = db.Begin()
		innerTrans = true
		if err != nil {
			return ret, err
		}
	}

	for k, v := range scriptParams {
		*script = strings.Replace(*script, k, v, -1)
	}

	for k, v := range replaceContext {
		*script = strings.Replace(*script, k, v, -1)
	}

	scriptsArray, err := gosplitargs.SplitArgs(*script, ";", true)
	if err != nil {
		if innerTrans {
			tx.Rollback()
		}
		return ret, err
	}
	for _, params1 := range params {
		totalCount := 0
		result := []interface{}{}
		for _, s := range scriptsArray {
			sqlNormalize(&s)
			if len(s) == 0 {
				continue
			}
			count, err := gosplitargs.CountSeparators(s, "\\?")
			if err != nil {
				if innerTrans {
					tx.Rollback()
				}
				return nil, err
			}
			if len(params1) < totalCount+count {
				if innerTrans {
					tx.Rollback()
				}
				return nil, errors.New(fmt.Sprintln("Incorrect param count. Expected: ", totalCount+count, " actual: ", len(params1)))
			}
			isQ := isQuery(s)
			if isQ {
				if array {
					header, data, err := gosqljson.QueryTxToArray(tx, theCase, s, params1[totalCount:totalCount+count]...)
					data = append([][]string{header}, data...)
					if err != nil {
						if innerTrans {
							tx.Rollback()
						}
						return nil, err
					}
					result = append(result, data)
				} else {
					data, err := gosqljson.QueryTxToMap(tx, theCase, s, params1[totalCount:totalCount+count]...)
					if err != nil {
						if innerTrans {
							tx.Rollback()
						}
						return nil, err
					}
					result = append(result, data)
				}
			} else {
				rowsAffected, err := gosqljson.ExecTx(tx, s, params1[totalCount:totalCount+count]...)
				if err != nil {
					if innerTrans {
						tx.Rollback()
					}
					return nil, err
				}
				result = append(result, rowsAffected)
			}

			totalCount += count
		}
		ret = append(ret, result)
	}

	if innerTrans {
		tx.Commit()
	}
	return ret, nil
}

func buildReplaceContext(context map[string]interface{}) map[string]string {
	replaceContext := map[string]string{}
	if clientIp, ok := context["client_ip"].(string); ok {
		replaceContext["__ip__"] = clientIp
	}
	if loginUserId, ok := context["user_id"].(string); ok {
		replaceContext["__login_user_id__"] = loginUserId
	}
	if loginUserCode, ok := context["email"].(string); ok {
		replaceContext["__login_user_code__"] = loginUserCode
	}
	if tokenUserId, ok := context["token_user_id"].(string); ok {
		replaceContext["__token_user_id__"] = tokenUserId
	}
	if tokenUserCode, ok := context["token_user_code"].(string); ok {
		replaceContext["__token_user_code__"] = tokenUserCode
	}
	return replaceContext
}

func buildParams(clientData string) (map[string]string, [][]interface{}, error) {
	// assume the clientData is a json object with two arrays: query_params and params
	parser, err := gojq.NewStringQuery(clientData)
	if err != nil {
		return nil, nil, err
	}
	qp, err := parser.QueryToMap("query_params")
	if err != nil {
		return nil, nil, err
	}
	queryParams, err := convertMapOfInterfacesToMapOfStrings(qp)
	if err != nil {
		return nil, nil, err
	}
	p, err := parser.Query("params")
	if p1, ok := p.([]interface{}); ok {
		params := [][]interface{}{}
		for _, p2 := range p1 {
			if param, ok := p2.([]interface{}); ok {
				params = append(params, param)
			} else {
				return nil, nil, errors.New("Failed to build params.")
			}
		}
		return queryParams, params, nil
	}
	return nil, nil, errors.New("Failed to build.")
}

func convertInterfaceArrayToStringArray(arrayOfInterfaces []interface{}) ([]string, error) {
	ret := []string{}
	for _, v := range arrayOfInterfaces {
		if s, ok := v.(string); ok {
			ret = append(ret, s)
		} else {
			return nil, errors.New("Failed to convert.")
		}
	}
	return ret, nil
}

var convertMapOfInterfacesToMapOfStrings = func(data map[string]interface{}) (map[string]string, error) {
	if data == nil {
		return nil, errors.New("Cannot convert nil.")
	}
	ret := map[string]string{}
	for k, v := range data {
		if v == nil {
			return nil, errors.New("Data contains nil.")
		}
		ret[k] = v.(string)
	}
	return ret, nil
}

func sqlNormalize(sql *string) {
	*sql = strings.TrimSpace(*sql)
	var ret string
	lines := strings.Split(*sql, "\n")
	for _, line := range lines {
		lineTrimmed := strings.TrimSpace(line)
		if lineTrimmed != "" && !strings.HasPrefix(lineTrimmed, "-- ") {
			ret += line + "\n"
		}
	}
	*sql = ret
}

func isQuery(sql string) bool {
	sqlUpper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(sqlUpper, "SELECT") ||
		strings.HasPrefix(sqlUpper, "SHOW") ||
		strings.HasPrefix(sqlUpper, "DESCRIBE") ||
		strings.HasPrefix(sqlUpper, "EXPLAIN") {
		return true
	}
	return false
}

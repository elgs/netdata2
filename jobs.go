// jobs
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/elgs/gorest2"
	"github.com/elgs/gosplitargs"
	"github.com/elgs/gosqljson"
)

func init() {
	jobModes["sql"] = func(job map[string]string) func() {
		return func() {
			defer func() {
				if err := recover(); err != nil {
					log.Println(err)
				}
			}()
			script := job["SCRIPT"]
			projectId := job["PROJECT_ID"]
			loopScript := job["LOOP_SCRIPT"]

			dbo, err := gorest2.GetDbo(projectId)
			if err != nil {
				log.Println(err)
				return
			}
			db, err := dbo.GetConn()
			if err != nil {
				log.Println(err)
				return
			}
			tx, err := db.Begin()
			if err != nil {
				log.Println(err)
				return
			}

			sqlNormalize(&loopScript)
			if len(loopScript) > 0 {
				_, loopData, err := gosqljson.QueryTxToArray(tx, "", loopScript)
				if err != nil {
					log.Println(err)
					tx.Rollback()
					return
				}
				for _, row := range loopData {
					scriptReplaced := script
					for i, v := range row {
						scriptReplaced = strings.Replace(script, fmt.Sprint("$", i), v, -1)
					}

					scriptsArray, err := gosplitargs.SplitArgs(scriptReplaced, ";", true)
					if err != nil {
						log.Println(err)
						tx.Rollback()
						return
					}

					for _, s := range scriptsArray {
						sqlNormalize(&s)
						if len(s) == 0 {
							continue
						}
						_, err = gosqljson.ExecTx(tx, s)
						if err != nil {
							tx.Rollback()
							log.Println(err)
							return
						}
					}
				}
			} else {
				scriptsArray, err := gosplitargs.SplitArgs(script, ";", true)
				if err != nil {
					log.Println(err)
					tx.Rollback()
					return
				}

				for _, s := range scriptsArray {
					sqlNormalize(&s)
					if len(s) == 0 {
						continue
					}
					_, err = gosqljson.ExecTx(tx, s)
					if err != nil {
						tx.Rollback()
						log.Println(err)
						return
					}
				}
			}
			tx.Commit()
		}
	}
}

var jobModes = make(map[string]func(map[string]string) func())
var jobStatus = make(map[string]int)

func OnJobCreate(app *Job) error { return nil }
func OnJobUpdate(app *Job) error { return nil }
func OnJobRemove(app *Job) error { return nil }

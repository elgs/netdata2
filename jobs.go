// jobs
package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/elgs/cron"
	"github.com/elgs/gorest2"
	"github.com/elgs/gosplitargs"
	"github.com/elgs/gosqljson"
)

func (this *Job) Action(mode string) func() {
	return func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
			}
		}()
		script := this.Script
		appId := this.AppId
		loopScript := this.LoopScript

		dbo, err := gorest2.GetDbo(appId)
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

var Sched *cron.Cron
var jobStatus = make(map[string]int)

func StartJobs() {
	Sched = cron.New()
	for _, app := range masterData.Apps {
		for _, job := range app.Jobs {
			if job.AutoStart {
				h, err := Sched.AddFunc(job.Cron, job.Action("sql"))
				if err != nil {
					log.Println(err)
					continue
				}
				jobStatus[job.Id] = h
			}
		}
	}
	Sched.Start()
}

func (this *Job) Start() error {
	if _, ok := jobStatus[this.Id]; ok {
		return errors.New("Job already started: " + this.Id)
	}
	jobRuntimeId, err := Sched.AddFunc(this.Cron, this.Action("sql"))
	if err != nil {
		return err
	}
	jobStatus[this.Id] = jobRuntimeId
	return nil
}
func (this *Job) Restart() error {
	err := this.Stop()
	if err != nil {
		return err
	}
	return this.Start()
}
func (this *Job) Stop() error {
	if jobRuntimeId, ok := jobStatus[this.Id]; ok {
		Sched.RemoveFunc(jobRuntimeId)
		delete(jobStatus, this.Id)
	} else {
		return errors.New("Job not started: " + this.Id)
	}
	return nil
}
func (this *Job) Started() bool {
	if _, ok := jobStatus[this.Id]; ok {
		return true
	} else {
		return false
	}
}

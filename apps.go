// apps
package main

import (
	"database/sql"
	"fmt"

	"github.com/elgs/gosqljson"
)

func OnAppCreateOrUpdate(app *App) error {
	ds := fmt.Sprintf("%v:%v@tcp(%v:%v)/", app.DataNode.Username, app.DataNode.Password,
		app.DataNode.Host, app.DataNode.Port)
	appDb, err := sql.Open("mysql", ds)
	defer appDb.Close()
	if err != nil {
		return err
	}

	_, err = gosqljson.ExecDb(appDb, "CREATE DATABASE IF NOT EXISTS nd_"+app.DbName+
		" DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci")
	if err != nil {
		return err
	}

	sqlGrant := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO `%s`@`%%` IDENTIFIED BY \"%s\";", "nd_"+app.DbName, app.DbName, app.Id)
	_, err = gosqljson.ExecDb(appDb, sqlGrant)
	if err != nil {
		return err
	}
	return nil
}

func OnAppRemove(app *App) error {

	var newQueries []Query
	for _, v := range masterData.Queries {
		if v.AppId != app.Id {
			newQueries = append(newQueries, v)
		}
	}
	masterData.Queries = newQueries

	var newJobs []Job
	for _, v := range masterData.Jobs {
		if v.AppId != app.Id {
			newJobs = append(newJobs, v)
		}
	}
	masterData.Jobs = newJobs

	var newTokens []Token
	for _, v := range masterData.Tokens {
		if v.AppId != app.Id {
			newTokens = append(newTokens, v)
		}
	}
	masterData.Tokens = newTokens

	var newLi []LocalInterceptor
	for _, v := range masterData.LocalInterceptors {
		if v.AppId != app.Id {
			newLi = append(newLi, v)
		}
	}
	masterData.LocalInterceptors = newLi

	var newRi []RemoteInterceptor
	for _, v := range masterData.RemoteInterceptors {
		if v.AppId != app.Id {
			newRi = append(newRi, v)
		}
	}
	masterData.RemoteInterceptors = newRi

	ds := fmt.Sprintf("%v:%v@tcp(%v:%v)/", app.DataNode.Username, app.DataNode.Password,
		app.DataNode.Host, app.DataNode.Port)
	appDb, err := sql.Open("mysql", ds)
	defer appDb.Close()
	if err != nil {
		return err
	}

	// Drop database
	_, err = gosqljson.ExecDb(appDb, "DROP DATABASE IF EXISTS nd_"+app.DbName)
	if err != nil {
		return err
	}

	sqlDropUser := fmt.Sprintf("DROP USER `%s`", app.DbName)
	_, err = gosqljson.ExecDb(appDb, sqlDropUser)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

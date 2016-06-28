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

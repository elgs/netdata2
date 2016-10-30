// apps
package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/elgs/gosqljson"
)

func (this *App) OnAppCreateOrUpdate() error {
	var dn *DataNode = nil
	for iDn, vDn := range masterData.DataNodes {
		if this.DataNodeId == vDn.Id {
			dn = masterData.DataNodes[iDn]
			break
		}
	}

	if dn == nil {
		return errors.New("Data node not found: " + this.DataNodeId)
	}

	ds := fmt.Sprintf("%v:%v@tcp(%v:%v)/", dn.Username, dn.Password, dn.Host, dn.Port)
	appDb, err := sql.Open("mysql", ds)
	defer appDb.Close()
	if err != nil {
		return err
	}

	_, err = gosqljson.ExecDb(appDb, "CREATE DATABASE IF NOT EXISTS nd_"+this.DbName+
		" DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci")
	if err != nil {
		return err
	}

	sqlGrant := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO `%s`@`%%` IDENTIFIED BY \"%s\";", "nd_"+this.DbName, this.DbName, this.Id)
	_, err = gosqljson.ExecDb(appDb, sqlGrant)
	if err != nil {
		return err
	}
	return nil
}

func (this *App) OnAppRemove() error {
	var dn *DataNode = nil
	for iDn, vDn := range masterData.DataNodes {
		if this.DataNodeId == vDn.Id {
			dn = masterData.DataNodes[iDn]
			break
		}
	}

	if dn == nil {
		return errors.New("Data node not found: " + this.DataNodeId)
	}

	ds := fmt.Sprintf("%v:%v@tcp(%v:%v)/", dn.Username, dn.Password, dn.Host, dn.Port)
	appDb, err := sql.Open("mysql", ds)
	defer appDb.Close()
	if err != nil {
		return err
	}

	// Drop database
	_, err = gosqljson.ExecDb(appDb, "DROP DATABASE IF EXISTS nd_"+this.DbName)
	if err != nil {
		return err
	}

	sqlDropUser := fmt.Sprintf("DROP USER IF EXISTS `%s`", this.DbName)
	_, err = gosqljson.ExecDb(appDb, sqlDropUser)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

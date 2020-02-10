package db

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/kvzoo/client"
	"github.com/zdnscloud/kvzoo/server"

	"github.com/zdnscloud/singlecloud/config"
)

const (
	DBFileName     = "singlecloud.db"
	DBVersionTable = "version"
	DBVersion      = "v1.0"
)

type Version struct {
	Version string `json:"version"`
}

var globalDB kvzoo.DB

func GetGlobalDB() kvzoo.DB {
    return globalDB
}

func RunAsMaster(conf *config.SinglecloudConf, stopCh chan struct{}) error {
	dbServerAddr := fmt.Sprintf(":%d", conf.DB.Port)
	db, err := server.NewWithBoltDB(dbServerAddr, path.Join(conf.DB.Path, DBFileName))
	if err != nil {
		return err
	}

	dbStarted := make(chan struct{})
	go func() {
		close(dbStarted)
		db.Start()
	}()
	<-dbStarted

	var slaves []string
	if conf.DB.SlaveDBAddr != "" {
		slaves = append(slaves, conf.DB.SlaveDBAddr)
	}

	globalDB, err = client.New(dbServerAddr, slaves)
	if err != nil {
		db.Stop()
		return err
	}

	go func() {
		<-stopCh
		db.Stop()
	}()

	if err := checkDBVersion(globalDB); err != nil {
		return err
	}

	if conf.DB.SlaveDBAddr != "" {
		if _, err := globalDB.Checksum(); err != nil {
			return err
		}
	}

	return nil
}

func checkDBVersion(db kvzoo.DB) error {
	tn, _ := kvzoo.TableNameFromSegments(DBVersionTable)
	table, err := db.CreateOrGetTable(tn)
	if err != nil {
		return fmt.Errorf("create or get table %s failed: %s", tn, err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin table %s transaction failed: %s", tn, err.Error())
	}

	defer tx.Rollback()
	values, err := tx.List()
	if err != nil {
		return fmt.Errorf("get db version failed: %s", err.Error())
	}

	if len(values) == 0 {
		v := &Version{Version: DBVersion}
		value, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal db version failed: %s", err.Error())
		}

		if err := tx.Add(v.Version, value); err != nil {
			return fmt.Errorf("add version to db failed: %s", err.Error())
		}

		log.Debugf("init db version with %s", DBVersion)
		return tx.Commit()
	}

	for v := range values {
		if v != DBVersion {
			return fmt.Errorf("invalid db version %s, current db version is %s", v, DBVersion)
		}
	}

	return nil
}

func RunAsSlave(conf *config.SinglecloudConf) {
	dbServerAddr := fmt.Sprintf(":%d", conf.DB.Port)
	db, err := server.NewWithBoltDB(dbServerAddr, path.Join(conf.DB.Path, DBFileName))
	if err != nil {
		log.Fatalf("start slave failed:%s", err.Error())
		return
	}

	db.Start()
}

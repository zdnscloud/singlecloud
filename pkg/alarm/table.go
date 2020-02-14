package alarm

import (
	"encoding/json"
	"fmt"

	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/db"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func genTable(name string) (kvzoo.Table, error) {
	tn, _ := kvzoo.TableNameFromSegments(name)
	table, err := db.GetGlobalDB().CreateOrGetTable(tn)
	if err != nil {
		return nil, fmt.Errorf("create or get table %s failed: %s", name, err.Error())
	}
	return table, nil
}

func addOrUpdateAlarmToDB(table kvzoo.Table, alarm *types.Alarm, action string) error {
	value, err := json.Marshal(alarm)
	if err != nil {
		return fmt.Errorf("marshal list %s failed: %s", alarm.UID, err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	defer tx.Rollback()
	switch action {
	case "add":
		if err = tx.Add(uintToStr(alarm.UID), value); err != nil {
			return err
		}
	case "update":
		if err = tx.Update(uintToStr(alarm.UID), value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func getAlarmsFromDB(table kvzoo.Table) ([]*types.Alarm, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()
	values, err := tx.List()
	if err != nil {
		return nil, err
	}
	alarms := make([]*types.Alarm, 0)
	for _, value := range values {
		var alarm types.Alarm
		if err := json.Unmarshal(value, &alarm); err != nil {
			return nil, err
		}
		alarms = append(alarms, &alarm)
	}
	return alarms, nil
}

func deleteAlarmFromDB(table kvzoo.Table, id string) error {
	tx, err := table.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := tx.Delete(id); err != nil {
		return err
	}

	return tx.Commit()
}

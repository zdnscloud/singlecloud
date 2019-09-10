package zke

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/types"
)

const (
	ZKEManagerDBTable = "zke_manager"
)

type clusterState struct {
	*core.FullState  `json:",inline"`
	*types.ZKEConfig `json:",inline"`
	CreateTime       time.Time `json:"createTime"`
	IsUnvailable     bool      `json:"isUnvailable"`
}

func getState(clusterID string, db storage.DB) (clusterState, error) {
	table, err := db.CreateOrGetTable(storage.GenTableName(ZKEManagerDBTable))
	if err != nil {
		return clusterState{}, fmt.Errorf("get table failed: %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return clusterState{}, fmt.Errorf("begin transaction failed: %s", err.Error())
	}
	defer tx.Commit()

	value, err := tx.Get(clusterID)

	if err != nil {
		return clusterState{}, fmt.Errorf("get cluster %s  state failed %s", clusterID, err.Error())
	}

	state, err := readStateJsonBytes(value)
	if err != nil {
		return state, fmt.Errorf("read cluster %s  state failed %s", clusterID, err.Error())
	}
	return state, nil
}

func createOrUpdateState(clsuterID string, s clusterState, db storage.DB) error {
	table, err := db.CreateOrGetTable(storage.GenTableName(ZKEManagerDBTable))
	if err != nil {
		return fmt.Errorf("get table failed %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed %s", err.Error())
	}
	defer tx.Rollback()

	value, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal cluster %s state failed %s", clsuterID, err.Error())
	}

	existValue, _ := tx.Get(clsuterID)
	if existValue == nil {
		if err := tx.Add(clsuterID, value); err != nil {
			return fmt.Errorf("add cluster %s state failed %s", clsuterID, err.Error())
		}
	} else {
		if err := tx.Update(clsuterID, value); err != nil {
			return fmt.Errorf("update cluster %s  state failed %s", clsuterID, err.Error())
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cluster %s  state failed %s", clsuterID, err.Error())
	}

	return nil
}

func deleteState(clusterID string, db storage.DB) error {
	table, err := db.CreateOrGetTable(storage.GenTableName(ZKEManagerDBTable))
	if err != nil {
		return fmt.Errorf("get table failed %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed %s", err.Error())
	}
	defer tx.Rollback()

	if err := tx.Delete(clusterID); err != nil {
		return fmt.Errorf("delete cluster %s  state failed %s", clusterID, err.Error())
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cluster %s  state failed %s", clusterID, err.Error())
	}

	return nil
}

func listState(db storage.DB) (map[string]clusterState, error) {
	stateMap := make(map[string]clusterState)

	table, err := db.CreateOrGetTable(storage.GenTableName(ZKEManagerDBTable))
	if err != nil {
		return stateMap, fmt.Errorf("get table failed %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return stateMap, fmt.Errorf("begin transaction failed %s", err.Error())
	}
	defer tx.Commit()

	values, err := tx.List()
	if err != nil {
		return stateMap, fmt.Errorf("list cluster state failed %s", err.Error())
	}

	for k, v := range values {
		state, err := readStateJsonBytes(v)
		if err != nil {
			return stateMap, fmt.Errorf("read cluster %s state failed %s", k, err.Error())
		}
		stateMap[k] = state
	}
	return stateMap, nil
}

func readStateJsonBytes(sj []byte) (clusterState, error) {
	s := clusterState{}
	if err := json.Unmarshal(sj, &s); err != nil {
		return s, err
	}
	if s.FullState != nil && s.FullState.DesiredState.CertificatesBundle != nil {
		s.DesiredState.CertificatesBundle = pki.TransformPEMToObject(s.DesiredState.CertificatesBundle)
		s.CurrentState.CertificatesBundle = pki.TransformPEMToObject(s.CurrentState.CertificatesBundle)
	}
	return s, nil
}

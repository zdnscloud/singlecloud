package handler

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

const RegistryCertManagerDBTable = "registry_certs"

type RegistryCertManager struct {
	api.DefaultHandler
	clusters *ClusterManager
	dbTable  *storage.DBTable
	lock     sync.Mutex
}

func newRegistryCertManager(clusterMgr *ClusterManager) *RegistryCertManager {
	return &RegistryCertManager{
		clusters: clusterMgr,
	}
}

func (m *RegistryCertManager) List(ctx *resttypes.Context) interface{} {
	m.lock.Lock()
	defer m.lock.Unlock()
	rcs, err := m.listCertFromDB()
	if err != nil {
		return nil
	}
	return rcs
}

func (m *RegistryCertManager) Create(ctx *resttypes.Context, yaml []byte) (interface{}, *resttypes.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	id := ctx.Object.GetID()
	rc, err := m.getCertFromDB(id)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	if rc != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("registrycert %s has exists", id))
	}

	inner := ctx.Object.(*types.RegistryCert)
	inner.SetCreationTimestamp(time.Now())
	if len(inner.Cert) == 0 || len(inner.Domain) == 0 {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("registrycert %s domain and cert content cat't be nil", id))
	}
	if err := m.addOrUpdateCertFromDB(inner); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	return inner, nil
}

func (m *RegistryCertManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	m.lock.Lock()
	defer m.lock.Unlock()
	id := ctx.Object.GetID()
	rc, _ := m.getCertFromDB(id)
	if rc == nil {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("registrycert %s does not exists", id))
	}
	if err := m.deleteCertFromDB(id); err != nil {
		return resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	return nil
}

func (m *RegistryCertManager) addOrUpdateCertFromDB(rc *types.RegistryCert) error {
	tx, err := m.dbTable.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed %s", err.Error())
	}
	defer tx.Rollback()

	value, err := json.Marshal(rc)
	if err != nil {
		return fmt.Errorf("marshal registrycert %s failed %s", rc.Domain, err.Error())
	}

	existValue, _ := tx.Get(rc.Domain)
	if existValue == nil {
		if err := tx.Add(rc.Domain, value); err != nil {
			return fmt.Errorf("add registrycert %s failed %s", rc.Domain, err.Error())
		}
	} else {
		if err := tx.Update(rc.Domain, value); err != nil {
			return fmt.Errorf("update registrycert %s failed %s", rc.Domain, err.Error())
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit registrycert %s failed %s", rc.Domain, err.Error())
	}

	return nil
}

func (m *RegistryCertManager) deleteCertFromDB(registryDomain string) error {
	tx, err := m.dbTable.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed %s", err.Error())
	}
	defer tx.Rollback()

	if err := tx.Delete(registryDomain); err != nil {
		return fmt.Errorf("delete registrycert %s failed %s", registryDomain, err.Error())
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit registrycert %s failed %s", registryDomain, err.Error())
	}

	return nil
}

func (m *RegistryCertManager) listCertFromDB() ([]*types.RegistryCert, error) {
	rcs := []*types.RegistryCert{}
	tx, err := m.dbTable.Begin()
	if err != nil {
		return rcs, fmt.Errorf("begin transaction failed %s", err.Error())
	}
	defer tx.Commit()

	values, err := tx.List()
	if err != nil {
		return rcs, fmt.Errorf("list registrycert failed %s", err.Error())
	}

	for _, v := range values {
		rc := &types.RegistryCert{}
		if err := json.Unmarshal(v, rc); err != nil {
			continue
		}
		rcs = append(rcs, rc)
	}
	return rcs, nil
}

func (m *RegistryCertManager) getCertFromDB(registryDomain string) (*types.RegistryCert, error) {
	tx, err := m.dbTable.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction failed %s", err.Error())
	}
	defer tx.Commit()

	value, err := tx.Get(registryDomain)
	if err != nil {
		return nil, fmt.Errorf("get registrycert failed %s", err.Error())
	}
	rc := &types.RegistryCert{}
	if err := json.Unmarshal(value, rc); err != nil {
		return nil, err
	}
	return rc, nil
}

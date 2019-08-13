package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cluster-agent/registrycert"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	RegistryCertManagerDBTable  = "registry_certs"
	ClusterAgentRegistryCertUrl = "/apis/agent.zcloud.cn/v1/registrycerts"
)

type RegistryCertManager struct {
	api.DefaultHandler
	clusters *ClusterManager
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
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create registry cert")
	}
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
	go m.deployCertToClusters(inner)
	if err := m.addOrUpdateCertFromDB(inner); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	return inner, nil
}

func (m *RegistryCertManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can update registry cert")
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	id := ctx.Object.GetID()
	rc, _ := m.getCertFromDB(id)
	if rc == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("registrycert %s does not exists", id))
	}
	inner := ctx.Object.(*types.RegistryCert)
	if inner.Domain != rc.Domain {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("registrycert %s domain cat't change", id))
	}
	if len(inner.Cert) == 0 {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("registrycert %s cert cat't be nil", id))
	}
	go m.deployCertToClusters(inner)
	if err := m.addOrUpdateCertFromDB(inner); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	return inner, nil
}

func (m *RegistryCertManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can delete global registry cert")
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	id := ctx.Object.GetID()
	rc, _ := m.getCertFromDB(id)
	if rc == nil {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("registrycert %s does not exists", id))
	}
	go m.deleteCertFromClusters(rc)
	if err := m.deleteCertFromDB(id); err != nil {
		return resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	return nil
}

func (m *RegistryCertManager) addOrUpdateCertFromDB(rc *types.RegistryCert) error {
	table, err := m.clusters.GetDB().CreateOrGetTable(RegistryCertManagerDBTable)
	tx, err := table.Begin()
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
	table, err := m.clusters.GetDB().CreateOrGetTable(RegistryCertManagerDBTable)
	tx, err := table.Begin()
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
	table, err := m.clusters.GetDB().CreateOrGetTable(RegistryCertManagerDBTable)
	tx, err := table.Begin()
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
	table, err := m.clusters.GetDB().CreateOrGetTable(RegistryCertManagerDBTable)
	tx, err := table.Begin()
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

func (m *RegistryCertManager) deployCertToClusters(rc *types.RegistryCert) {
	for cluster, _ := range rc.Clusters {
		go deployCertToOneCluster(rc, cluster, m.clusters.Agent)
	}
}

func deployCertToOneCluster(rc *types.RegistryCert, cluster string, agent *clusteragent.AgentManager) error {
	deployRc := &registrycert.RegistryCert{
		Domain: rc.Domain,
		Cert:   rc.Cert,
	}
	requestBody, err := json.Marshal(deployRc)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", ClusterAgentRegistryCertUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := agent.ProxyRequest(cluster, req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (m *RegistryCertManager) deleteCertFromClusters(rc *types.RegistryCert) {
	for cluster, _ := range rc.Clusters {
		go deleteCertFromOneCluster(rc.Domain, cluster, m.clusters.Agent)
	}
}

func deleteCertFromOneCluster(id string, cluster string, agent *clusteragent.AgentManager) error {
	req, err := http.NewRequest("DELETE", ClusterAgentRegistryCertUrl+"/"+id, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := agent.ProxyRequest(cluster, req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

package acl

import (
	"net"
	"strings"
	"sync"

	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/httpcmd"
	"github.com/zdnscloud/vanguard/logger"
)

const (
	AnyAcl  string = "any"
	NoneAcl        = "none"
	AllAcl         = "all"
	BadName        = "acl"
)

type AclManager struct {
	acls      map[string]*Acl
	scheduler *AclScheduler
	lock      sync.RWMutex
}

var gAcl *AclManager

func NewAclManager(conf *config.VanguardConf) {
	gAcl = &AclManager{}

	gAcl.ReloadConfig(conf)
	httpcmd.RegisterHandler(gAcl, []httpcmd.Command{&AddAcl{}, &DeleteAcl{}, &UpdateAcl{}})
}

func GetAclManager() *AclManager {
	return gAcl
}

func (m *AclManager) ReloadConfig(conf *config.VanguardConf) {
	scheduler := NewAclScheduler()
	go scheduler.Run()

	aclMap := make(map[string]*Acl)
	for _, a := range conf.Acls {
		acl, err := NewAcl(a.Networks.IPs, a.Networks.ValidInterval, a.Networks.InvalidInterval)
		if err != nil {
			panic("load acl " + a.Name + " failed " + err.Error())
		}

		aclMap[a.Name] = acl
		scheduler.Add(acl)
	}

	m.acls = aclMap
	m.scheduler = scheduler
}

func (m *AclManager) Stop() {
	m.scheduler.Stop()
}

func (m *AclManager) Find(aclName string, ip net.IP) bool {
	lowerAcl := strings.ToLower(aclName)
	if lowerAcl == AnyAcl {
		return true
	} else if lowerAcl == NoneAcl {
		return false
	}

	m.lock.RLock()
	acl, ok := m.acls[aclName]
	m.lock.RUnlock()
	if ok == false {
		logger.GetLogger().Warn("acl %s is no exist", aclName)
		return false
	}

	return acl.Include(ip)
}

func (m *AclManager) add(aclName string, ips []string) *httpcmd.Error {
	acl, err := NewAcl(ips, nil, nil)
	if err != nil {
		return httpcmd.ErrInvalidNetwork.AddDetail(err.Error())
	}

	m.lock.Lock()
	m.acls[aclName] = acl
	m.lock.Unlock()
	m.scheduler.Add(acl)

	return nil
}

func (m *AclManager) remove(aclName string) *httpcmd.Error {
	acl, ok := m.acls[aclName]
	if ok == false {
		return ErrNonExistAcl
	}

	m.lock.Lock()
	delete(m.acls, aclName)
	m.lock.Unlock()
	m.scheduler.Delete(acl)
	return nil
}

func (m *AclManager) update(aclName string, ips []string) *httpcmd.Error {
	if err := m.remove(aclName); err != nil {
		return err
	} else {
		return m.add(aclName, ips)
	}
}

func (m *AclManager) hasAcl(aclName string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.acls[aclName]
	return ok
}

package acl

import (
	"time"
)

const (
	DefaultCheckInterval = time.Minute
)

type AclScheduler struct {
	addAclChan    chan *Acl
	deleteAclChan chan *Acl
	stopChan      chan struct{}
}

func NewAclScheduler() *AclScheduler {
	return &AclScheduler{
		addAclChan:    make(chan *Acl),
		deleteAclChan: make(chan *Acl),
		stopChan:      make(chan struct{}),
	}
}

func (s *AclScheduler) Run() {
	var acls []*Acl
	timer := time.NewTicker(DefaultCheckInterval)
	defer timer.Stop()

	for {
		select {
		case <-s.stopChan:
			s.stopChan <- struct{}{}
			return
		case newAcl := <-s.addAclChan:
			acls = append(acls, newAcl)
		case oldAcl := <-s.deleteAclChan:
			for i, acl := range acls {
				if acl == oldAcl {
					acls = append(acls[:i], acls[i+1:]...)
					break
				}
			}
		case <-timer.C:
			now := time.Now()
			for _, acl := range acls {
				acl.CheckValid(now)
			}
		}
	}
}

func (s *AclScheduler) Stop() {
	s.stopChan <- struct{}{}
	<-s.stopChan
}

func (s *AclScheduler) Add(acl *Acl) {
	s.addAclChan <- acl
}

func (s *AclScheduler) Delete(acl *Acl) {
	s.deleteAclChan <- acl
}

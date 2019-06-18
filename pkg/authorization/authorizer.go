package authorization

import (
	"fmt"
	"sync"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	Administrator string = "admin"
	AllNamespaces string = "_all_namespaces"
	AllClusters   string = "_all_clusters"
)

type Projects []types.Project

type Authorizer struct {
	users map[string]Projects
	lock  sync.RWMutex
}

func New() *Authorizer {
	return &Authorizer{
		users: make(map[string]Projects),
	}
}

func (m *Authorizer) Authorize(user, cluster, namespace string) bool {
	if user == Administrator {
		return true
	}

	m.lock.RLock()
	projects, ok := m.users[user]
	m.lock.RUnlock()
	if ok == false {
		return false
	}

	for _, project := range projects {
		if (project.Cluster == AllClusters || project.Cluster == cluster) &&
			(namespace == "" || project.Namespace == AllNamespaces || project.Namespace == namespace) {
			return true
		}
	}

	return false
}

func (m *Authorizer) AddUser(user string, projects []types.Project) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.users[user]; ok {
		return fmt.Errorf("user %s already exists", user)
	} else {
		m.users[user] = projects
		return nil
	}
}

func (m *Authorizer) DeleteUser(user string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.users[user]; ok {
		delete(m.users, user)
		return nil
	} else {
		return fmt.Errorf("user %s doesn't exist", user)
	}
}

func (m *Authorizer) UpdateUser(user string, projects []types.Project) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.users[user]; ok == false {
		return fmt.Errorf("user %s doesn't exist", user)
	} else {
		m.users[user] = projects
		return nil
	}
}

func (m *Authorizer) GetUserProjects(user string) []types.Project {
	m.lock.RLock()
	projects, ok := m.users[user]
	m.lock.RUnlock()

	if ok {
		return projects
	} else {
		return nil
	}
}

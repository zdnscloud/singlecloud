package authorization

import (
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AdminPasswd   string = "6710fc5dd8cd10e010af0083d9573fd327e8e67e" //hex encoding for sha1(zdns)
	AllNamespaces string = "_all_namespaces"
	AllClusters   string = "_all_clusters"
)

var adminUser = &types.User{
	Name: types.Administrator,
	Projects: []types.Project{
		types.Project{
			Cluster:   AllClusters,
			Namespace: AllNamespaces,
		},
	},
}

type Projects []types.Project

type Authorizer struct {
	users map[string]Projects
	lock  sync.RWMutex
}

func New() *Authorizer {
	auth := &Authorizer{
		users: make(map[string]Projects),
	}

	adminUser.SetID(types.Administrator)
	adminUser.SetType(types.UserType)
	adminUser.SetCreationTimestamp(time.Now())
	auth.AddUser(adminUser)
	return auth
}

func (m *Authorizer) Authorize(userName, cluster, namespace string) bool {
	if userName == types.Administrator {
		return true
	}

	m.lock.RLock()
	projects, ok := m.users[userName]
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

func (m *Authorizer) AddUser(user *types.User) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.users[user.Name]; ok {
		return fmt.Errorf("user %s already exists", user)
	} else {
		m.users[user.Name] = user.Projects
		return nil
	}
}

func (m *Authorizer) GetUser(userName string) *types.User {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if projects, ok := m.users[userName]; ok {
		return &types.User{
			Name:     userName,
			Projects: projects,
		}
	} else {
		return nil
	}
}

func (m *Authorizer) ListUser() []*types.User {
	m.lock.RLock()
	defer m.lock.RUnlock()
	users := make([]*types.User, 0, len(m.users))
	for name, projects := range m.users {
		users = append(users, &types.User{
			Name:     name,
			Projects: projects,
		})
	}
	return users
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

func (m *Authorizer) UpdateUser(user *types.User) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.users[user.Name]; ok == false {
		return fmt.Errorf("user %s doesn't exist", user)
	} else {
		m.users[user.Name] = user.Projects
		return nil
	}
}

package authorization

import (
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	Administrator string = "admin"
	AdminPasswd   string = "6710fc5dd8cd10e010af0083d9573fd327e8e67e" //hex encoding for sha1(zdns)
	AllNamespaces string = "_all_namespaces"
	AllClusters   string = "_all_clusters"
)

var adminUser = &types.User{
	Name:     Administrator,
	Password: AdminPasswd,
}

type Authorizer struct {
	users map[string]*types.User
	lock  sync.RWMutex
}

func New() *Authorizer {
	users := make(map[string]*types.User)
	adminUser.SetID(Administrator)
	adminUser.SetType(types.UserType)
	adminUser.SetCreationTimestamp(time.Now())
	users[Administrator] = adminUser
	return &Authorizer{
		users: users,
	}
}

func (m *Authorizer) Authorize(userName, cluster, namespace string) bool {
	if userName == Administrator {
		return true
	}

	m.lock.RLock()
	user, ok := m.users[userName]
	m.lock.RUnlock()
	if ok == false {
		return false
	}

	for _, project := range user.Projects {
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
		m.users[user.Name] = user
		return nil
	}
}

func (m *Authorizer) GetUser(userName string) *types.User {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.users[userName]
}

func (m *Authorizer) ListUser() []*types.User {
	m.lock.RLock()
	defer m.lock.RUnlock()
	users := make([]*types.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
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
		m.users[user.Name] = user
		return nil
	}
}

func (m *Authorizer) ResetPassword(userName string, old, new string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if user, ok := m.users[userName]; ok {
		if user.Password != old {
			return fmt.Errorf("password isn't correct")
		}
		user.Password = new
		return nil
	} else {
		return fmt.Errorf("user %s doesn't exist", userName)
	}
}

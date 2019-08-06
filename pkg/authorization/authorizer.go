package authorization

import (
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"
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
	db    storage.Table
}

func New(db storage.DB) (*Authorizer, error) {
	auth := &Authorizer{
		users: make(map[string]Projects),
	}

	if err := auth.loadUsers(db); err != nil {
		return nil, err
	}

	if _, ok := auth.users[types.Administrator]; ok == false {
		adminUser.SetID(types.Administrator)
		adminUser.SetType(types.UserType)
		adminUser.SetCreationTimestamp(time.Now())
		auth.AddUser(adminUser)
	}

	return auth, nil
}

func (a *Authorizer) Authorize(userName, cluster, namespace string) bool {
	if userName == types.Administrator {
		return true
	}

	a.lock.RLock()
	projects, ok := a.users[userName]
	a.lock.RUnlock()
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

func (a *Authorizer) AddUser(user *types.User) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	name := user.GetID()
	if _, ok := a.users[name]; ok {
		return fmt.Errorf("user %s already exists", name)
	} else {
		if err := a.addUser(user); err != nil {
			return err
		}
		a.users[name] = user.Projects
		return nil
	}
}

func (a *Authorizer) HasUser(userName string) bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	_, ok := a.users[userName]
	return ok
}

func (a *Authorizer) GetUser(userName string) *types.User {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if projects, ok := a.users[userName]; ok {
		user := &types.User{
			Name:     userName,
			Projects: projects,
		}
		user.SetID(userName)
		return user
	} else {
		return nil
	}
}

func (a *Authorizer) ListUser() []*types.User {
	a.lock.RLock()
	defer a.lock.RUnlock()
	users := make([]*types.User, 0, len(a.users))
	for name, projects := range a.users {
		user := &types.User{
			Name:     name,
			Projects: projects,
		}
		user.SetID(name)
		users = append(users, user)
	}
	return users
}

func (a *Authorizer) DeleteUser(user string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if _, ok := a.users[user]; ok {
		if err := a.deleteUser(user); err != nil {
			return err
		}
		delete(a.users, user)
		return nil
	} else {
		return fmt.Errorf("user %s doesn't exist", user)
	}
}

func (a *Authorizer) UpdateUser(user *types.User) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	name := user.GetID()
	if _, ok := a.users[name]; ok == false {
		return fmt.Errorf("user %s doesn't exist", name)
	} else {
		if err := a.updateUser(user); err != nil {
			return err
		}

		a.users[name] = user.Projects
		return nil
	}
}

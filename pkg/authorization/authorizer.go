package authorization

import (
	"fmt"
	"sync"
	"time"

	resttypes "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AdminPasswd   string = "6710fc5dd8cd10e010af0083d9573fd327e8e67e" //hex encoding for sha1(zdns)
	AllNamespaces string = "_all_namespaces"
	AllClusters   string = "_all_clusters"
)

var (
	zcloudStartTime, _ = time.Parse(time.RFC3339, "2019-02-28T00:00:00Z")
	adminUser          = &types.User{
		Name: types.Administrator,
		Projects: []types.Project{
			types.Project{
				Cluster:   AllClusters,
				Namespace: AllNamespaces,
			},
		},
	}
)

type User struct {
	Projects          []types.Project   `json:"projects,omitempty"`
	CreationTimestamp resttypes.ISOTime `json:"creationTimestamp,omitempty"`
	DeletionTimestamp resttypes.ISOTime `json:"deletionTimestamp,omitempty"`
}

type Authorizer struct {
	users map[string]*User
	lock  sync.RWMutex
	db    kvzoo.Table
}

func New(db kvzoo.DB) (*Authorizer, error) {
	auth := &Authorizer{
		users: make(map[string]*User),
	}

	if err := auth.loadUsers(db); err != nil {
		return nil, err
	}

	if _, ok := auth.users[types.Administrator]; ok == false {
		adminUser.SetID(types.Administrator)
		adminUser.SetCreationTimestamp(zcloudStartTime)
		auth.AddUser(adminUser)
	}

	return auth, nil
}

func (a *Authorizer) Authorize(userName, cluster, namespace string) bool {
	if userName == types.Administrator {
		return true
	}

	a.lock.RLock()
	user, ok := a.users[userName]
	a.lock.RUnlock()
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

func (a *Authorizer) AddUser(user *types.User) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	name := user.GetID()
	if _, ok := a.users[name]; ok {
		return fmt.Errorf("user %s already exists", name)
	} else {
		user_ := &User{
			Projects:          user.Projects,
			CreationTimestamp: resttypes.ISOTime(user.GetCreationTimestamp()),
		}
		if err := a.addUser(name, user_); err != nil {
			return err
		}
		a.users[name] = user_
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
	if user_, ok := a.users[userName]; ok {
		user := &types.User{
			Name:     userName,
			Projects: user_.Projects,
		}
		user.SetID(userName)
		user.SetCreationTimestamp(time.Time(user_.CreationTimestamp))
		user.SetDeletionTimestamp(time.Time(user_.DeletionTimestamp))
		return user
	} else {
		return nil
	}
}

func (a *Authorizer) ListUser() []*types.User {
	a.lock.RLock()
	defer a.lock.RUnlock()
	users := make([]*types.User, 0, len(a.users))
	for name, user_ := range a.users {
		user := &types.User{
			Name:     name,
			Projects: user_.Projects,
		}
		user.SetID(name)
		user.SetCreationTimestamp(time.Time(user_.CreationTimestamp))
		user.SetDeletionTimestamp(time.Time(user_.DeletionTimestamp))
		users = append(users, user)
	}
	return users
}

func (a *Authorizer) DeleteUser(user string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if user_, ok := a.users[user]; ok {
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
	if user_, ok := a.users[name]; ok == false {
		return fmt.Errorf("user %s doesn't exist", name)
	} else {
		user_.Projects = user.Projects
		if err := a.updateUser(name, user_); err != nil {
			return err
		}
		return nil
	}
}

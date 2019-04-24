package handler

import (
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authorize"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func getCurrentUser(ctx *resttypes.Context) *types.User {
	currentUser_, _ := ctx.Get(types.CurrentUserKey)
	return currentUser_.(*types.User)
}

func isAdmin(user *types.User) bool {
	return user.Name == authorize.Administrator
}

func hasClusterPermission(user *types.User, cluster string) bool {
	if isAdmin(user) {
		return true
	}

	for _, project := range user.Projects {
		if project.Cluster == cluster {
			return true
		}
	}
	return false
}

func hasNamespacePermission(user *types.User, cluster, namespace string) bool {
	if isAdmin(user) {
		return true
	}

	if user.Name == authorize.Administrator {
		return true
	}

	for _, project := range user.Projects {
		if project.Cluster == cluster && (project.Namespace == authorize.AllNamespace || project.Namespace == namespace) {
			return true
		}
	}
	return false
}

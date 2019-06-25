package handler

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/model"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type UserQuotaManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newUserQuotaManager(clusters *ClusterManager) *UserQuotaManager {
	return &UserQuotaManager{clusters: clusters}
}

func (m *UserQuotaManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	userName := getCurrentUser(ctx).Name
	if userName == "" {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, "create user quota failed: user name should not be empty")
	}

	userQuota := ctx.Object.(*types.UserQuota)
	exits, err := model.IsExistsNamespaceInDB(userQuota.Namespace)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("check exist for user %s namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	if exits {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource,
			fmt.Sprintf("duplicate namespace %s for user %s", userQuota.Namespace, userName))
	}

	setUserQuota(userQuota, userName, types.TypeUserQuotaCreate)
	userResourceQuotaId, err := model.SaveUserResourceQuotaToDB(scUserQuotaToDBUserResourceQuota(userQuota))
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("create user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}
	userQuota.SetID(userResourceQuotaId)
	return userQuota, nil
}

func (m *UserQuotaManager) List(ctx *resttypes.Context) interface{} {
	userResourceQuotas, err := model.GetUserResourceQuotasFromDB()
	if err != nil {
		log.Warnf("list user quota info failed: %s", err.Error())
		return nil
	}

	var userQuotas []*types.UserQuota
	for _, quota := range userResourceQuotas {
		userQuotas = append(userQuotas, dbUserResourceQuotaToSCUserQuota(&quota))
	}

	return userQuotas
}

func (m *UserQuotaManager) Get(ctx *resttypes.Context) interface{} {
	userQuota := ctx.Object.(*types.UserQuota)
	userResourceQuota, err := model.GetUserResourceQuotaByIDFromDB(userQuota.GetID())
	if err != nil {
		log.Warnf("get user quota %s info failed: %s", userQuota.GetID(), err.Error())
		return nil
	}

	return dbUserResourceQuotaToSCUserQuota(userResourceQuota)
}

func (m *UserQuotaManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	userName := getCurrentUser(ctx).Name
	if userName == "" {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, "update user quota failed: user name should not be empty")
	}

	userQuota := ctx.Object.(*types.UserQuota)
	userResourceQuota, err := model.GetUserResourceQuotaByIDFromDB(userQuota.GetID())
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	if userResourceQuota.UserName != userName {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("user %s can`t update quota which belong to %s", userName, userResourceQuota.UserName))
	}

	if userResourceQuota.Status == types.StatusUserQuotaProcessing {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("can`t update user quota which status is processing"))
	}

	setUserQuota(userQuota, userName, types.TypeUserQuotaUpdate)
	if err := model.UpdateUserResourceQuotaToDB(scUserQuotaToDBUserResourceQuota(userQuota)); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	return userQuota, nil
}

func (m *UserQuotaManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	user := getCurrentUser(ctx)
	userName := user.Name
	if userName == "" {
		return resttypes.NewAPIError(types.ConnectClusterFailed, "update user quota failed: user name should not be empty")
	}

	userQuota := ctx.Object.(*types.UserQuota)
	userResourceQuota, err := model.GetUserResourceQuotaByIDFromDB(userQuota.GetID())
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	if isAdmin(user) == false && userResourceQuota.UserName != userName {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("user %s can`t delete quota which belong to %s", userName, userResourceQuota.UserName))
	}

	if userResourceQuota.Status == types.StatusUserQuotaProcessing {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("can`t delete user quota which status is processing"))
	}

	if userResourceQuota.ClusterName != "" {
		cluster := m.clusters.GetClusterByName(userResourceQuota.ClusterName)
		if cluster == nil {
			return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
		}

		if err := deleteNamespace(cluster.KubeClient, userResourceQuota.Namespace); err != nil && apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete namespace failed %s", err.Error()))
		}
	}

	if err := model.DeleteUserResourceQuotaFromDB(userResourceQuota.Id); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete user quota failed: %v", err.Error()))
	}

	return nil
}

func (m *UserQuotaManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can approval or reject user quota")
	}

	switch ctx.Action.Name {
	case types.ActionApproval:
		return nil, m.approval(ctx)
	case types.ActionRejection:
		return nil, m.reject(ctx)
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
	}
}

func (m *UserQuotaManager) approval(ctx *resttypes.Context) *resttypes.APIError {
	clusterInfo, ok := ctx.Action.Input.(*types.ClusterInfo)
	if ok == false {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "approval param is not valid")
	}

	userQuotaID := ctx.Object.(*types.UserQuota).GetID()
	userResourceQuota, err := model.GetUserResourceQuotaByIDFromDB(userQuotaID)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user quota %s failed %s", userQuotaID, err.Error()))
	}

	if userResourceQuota.Status != types.StatusUserQuotaProcessing {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user quota %s failed: only approval request that status is processing", userQuotaID))
	}

	cluster := m.clusters.GetClusterByName(clusterInfo.ClusterName)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	var k8sResourceQuota *corev1.ResourceQuota
	var k8sResourceList corev1.ResourceList
	resourceQuota := &types.ResourceQuota{
		Name: userResourceQuota.Namespace,
		Limits: map[string]string{
			"limits.cpu":       userResourceQuota.CPU,
			"limits.memory":    userResourceQuota.Memory,
			"requests.storage": userResourceQuota.Storage,
		},
	}

	exists, err := hasNamespace(cluster.KubeClient, userResourceQuota.Namespace)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("check user %s namespace %s whether exists failed %s",
				userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
	}

	if exists == false {
		if err := createNamespace(cluster.KubeClient, userResourceQuota.Namespace); err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("create user %s namespace %s failed %s",
					userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
		}

		if err := createResourceQuota(cluster.KubeClient, userResourceQuota.Namespace, resourceQuota); err != nil {
			deleteNamespace(cluster.KubeClient, userResourceQuota.Namespace)
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("create user %s resourcequota with namespace %s failed %s",
					userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
		}
	} else {
		k8sResourceQuota, err = getResourceQuota(cluster.KubeClient, userResourceQuota.Namespace, userResourceQuota.Namespace)
		if err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("update user %s resourcequota with namespace %s failed %s",
					userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
		}
		k8sHard, err := scQuotaResourceListToK8sResourceList(resourceQuota.Limits)
		if err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("update user %s resourcequota with namespace %s failed %s",
					userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
		}
		k8sResourceList = k8sResourceQuota.Spec.Hard
		k8sResourceQuota.Spec.Hard = k8sHard
		if err := cluster.KubeClient.Update(context.TODO(), k8sResourceQuota); err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("update user %s resourcequota with namespace %s failed %s",
					userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
		}
	}

	userResourceQuota.Status = types.StatusUserQuotaApproval
	userResourceQuota.ClusterName = clusterInfo.ClusterName
	userResourceQuota.ResponseTimestamp = time.Now()
	if err := model.UpdateUserResourceQuotaToDB(userResourceQuota); err != nil {
		if exists == false {
			deleteNamespace(cluster.KubeClient, userResourceQuota.Namespace)
		} else {
			k8sResourceQuota.Spec.Hard = k8sResourceList
			cluster.KubeClient.Update(context.TODO(), k8sResourceQuota)
		}
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user %s quota with namespace %s failed %s",
				userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
	}
	return nil
}

func (m *UserQuotaManager) reject(ctx *resttypes.Context) *resttypes.APIError {
	rejection, ok := ctx.Action.Input.(*types.Rejection)
	if ok == false {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "rejection param is not valid")
	}

	userQuotaID := ctx.Object.(*types.UserQuota).GetID()
	userResourceQuota, err := model.GetUserResourceQuotaByIDFromDB(userQuotaID)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user quota %s failed: %s", userQuotaID, err.Error()))
	}

	if userResourceQuota.Status != types.StatusUserQuotaProcessing {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user quota %s failed: only reject request that status is processing", userQuotaID))
	}

	userResourceQuota.RejectionReason = rejection.Reason
	userResourceQuota.Status = types.StatusUserQuotaRejection
	userResourceQuota.ResponseTimestamp = time.Now()
	if err := model.UpdateUserResourceQuotaToDB(userResourceQuota); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user %s quota with namespace %s failed %s",
				userResourceQuota.UserName, userResourceQuota.Namespace, err.Error()))
	}
	return nil
}

func setUserQuota(userQuota *types.UserQuota, userName, requestType string) {
	userQuota.SetType(types.UserQuotaType)
	userQuota.SetCreationTimestamp(time.Now())
	userQuota.Status = types.StatusUserQuotaProcessing
	userQuota.UserName = userName
	userQuota.RequestType = requestType
}

func scUserQuotaToDBUserResourceQuota(quota *types.UserQuota) *model.UserResourceQuota {
	return &model.UserResourceQuota{
		Id:                quota.GetID(),
		ClusterName:       quota.ClusterName,
		Namespace:         quota.Namespace,
		UserName:          quota.UserName,
		CPU:               quota.CPU,
		Memory:            quota.Memory,
		Storage:           quota.Storage,
		RequestType:       quota.RequestType,
		Status:            quota.Status,
		Purpose:           quota.Purpose,
		CreationTimestamp: time.Time(quota.CreationTimestamp),
		Requestor:         quota.Requestor,
		Telephone:         quota.Telephone,
	}
}

func dbUserResourceQuotaToSCUserQuota(quota *model.UserResourceQuota) *types.UserQuota {
	userQuota := &types.UserQuota{
		ClusterName:       quota.ClusterName,
		Namespace:         quota.Namespace,
		UserName:          quota.UserName,
		CPU:               quota.CPU,
		Memory:            quota.Memory,
		Storage:           quota.Storage,
		RequestType:       quota.RequestType,
		Status:            quota.Status,
		Purpose:           quota.Purpose,
		RejectionReason:   quota.RejectionReason,
		ResponseTimestamp: resttypes.ISOTime(quota.ResponseTimestamp),
		Requestor:         quota.Requestor,
		Telephone:         quota.Telephone,
	}

	userQuota.SetID(quota.Id)
	userQuota.SetType(types.UserQuotaType)
	userQuota.SetCreationTimestamp(quota.CreationTimestamp)
	return userQuota
}

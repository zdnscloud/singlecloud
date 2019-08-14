package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"
)

const UserQuotaTable = "user_resource_quota"

type UserQuotaManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newUserQuotaManager(clusters *ClusterManager) *UserQuotaManager {
	return &UserQuotaManager{clusters: clusters}
}

func (m *UserQuotaManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	userName := getCurrentUser(ctx)
	if userName == "" {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, "create user quota failed: user name should not be empty")
	}

	userQuota := ctx.Object.(*types.UserQuota)
	if err := checkUserQuotaParamsValid(userQuota); err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("params is invalid: %s", err.Error()))
	}

	setUserQuota(userQuota, userName, types.TypeUserQuotaCreate, time.Now())
	value, err := json.Marshal(userQuota)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("marshal user quota to storage value failed: %s", err.Error()))
	}

	tx, err := BeginTableTransaction(m.clusters.GetDB(), UserQuotaTable)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("create user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	defer tx.Rollback()
	if err := tx.Add(userQuota.GetID(), value); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("create user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	if err := tx.Commit(); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("commit user_resource_quota table failed: %s", err.Error()))
	}

	return userQuota, nil
}

func (m *UserQuotaManager) List(ctx *resttypes.Context) interface{} {
	userName := getCurrentUser(ctx)
	tx, err := BeginTableTransaction(m.clusters.GetDB(), UserQuotaTable)
	if err != nil {
		log.Warnf("list user quota info failed: %s", err.Error())
		return nil
	}

	defer tx.Commit()
	values, err := tx.List()
	if err != nil {
		log.Warnf("list user quota info failed: %s", err.Error())
		return nil
	}

	var userQuotas types.UserQuotas
	for _, value := range values {
		quota, err := storageResourceValueToSCUserQuota(value)
		if err != nil {
			log.Warnf("list user quota info when unmarshal resource value failed: %s", err.Error())
			return nil
		}

		if isAdmin(userName) == false && quota.UserName != userName {
			continue
		}

		userQuotas = append(userQuotas, quota)
	}

	sort.Sort(userQuotas)
	return userQuotas
}

func (m *UserQuotaManager) Get(ctx *resttypes.Context) interface{} {
	userName := getCurrentUser(ctx)
	userQuota := ctx.Object.(*types.UserQuota)
	tx, err := BeginTableTransaction(m.clusters.GetDB(), UserQuotaTable)
	if err != nil {
		log.Warnf("get user quota info failed: %s", err.Error())
		return nil
	}

	defer tx.Commit()
	quota, err := getUserQuotaFromDB(tx, userQuota.GetID())
	if err != nil {
		log.Warnf("get user quota info failed: %s", err.Error())
		return nil
	}

	if isAdmin(userName) == false && quota.UserName != userName {
		log.Warnf("no found user quota %s for user %s", userQuota.GetID(), userName)
		return nil
	}

	return quota
}

func (m *UserQuotaManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	userName := getCurrentUser(ctx)
	if userName == "" {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, "update user quota failed: user name should not be empty")
	}

	userQuota := ctx.Object.(*types.UserQuota)
	if err := checkUserQuotaParamsValid(userQuota); err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("params is invalid: %s", err.Error()))
	}

	tx, err := BeginTableTransaction(m.clusters.GetDB(), UserQuotaTable)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	defer tx.Rollback()
	quota, err := getUserQuotaFromDB(tx, userQuota.GetID())
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	if quota.UserName != userName {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("user %s can`t update quota which belong to %s", userName, quota.UserName))
	}

	if quota.Status == types.StatusUserQuotaProcessing {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("can`t update user quota which status is processing"))
	}

	if quota.Namespace != userQuota.Namespace || quota.ClusterName != userQuota.ClusterName {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("can`t update namespace or clusterName for quota"))
	}

	setUserQuota(userQuota, userName, types.TypeUserQuotaUpdate, quota.GetCreationTimestamp())
	value, err := json.Marshal(userQuota)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("marshal user quota to storage value failed: %s", err.Error()))
	}

	if err := tx.Update(userQuota.GetID(), value); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	if err := tx.Commit(); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("commit user_resource_quota table failed: %s", err.Error()))
	}

	return userQuota, nil
}

func (m *UserQuotaManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	userName := getCurrentUser(ctx)
	if userName == "" {
		return resttypes.NewAPIError(types.ConnectClusterFailed, "update user quota failed: user name should not be empty")
	}

	userQuota := ctx.Object.(*types.UserQuota)
	tx, err := BeginTableTransaction(m.clusters.GetDB(), UserQuotaTable)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	defer tx.Rollback()
	quota, err := getUserQuotaFromDB(tx, userQuota.GetID())
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete user %s quota with namespace %s failed %s", userName, userQuota.Namespace, err.Error()))
	}

	if isAdmin(userName) == false && quota.UserName != userName {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("user %s can`t delete quota which belong to %s", userName, quota.UserName))
	}

	if quota.Status == types.StatusUserQuotaProcessing {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("can`t delete user quota which status is processing"))
	}

	if quota.ClusterName != "" {
		cluster := m.clusters.GetClusterByName(quota.ClusterName)
		if cluster == nil {
			return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
		}

		if err := deleteNamespace(cluster.KubeClient, quota.Namespace); err != nil && apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete namespace failed %s", err.Error()))
		}
	}

	if err := tx.Delete(userQuota.GetID()); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete user quota failed: %v", err.Error()))
	}

	if err := tx.Commit(); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete user quota failed: %v", err.Error()))
	}

	if quota.ClusterName != "" {
		authorizer := m.clusters.GetAuthorizer()
		user := authorizer.GetUser(quota.UserName)
		if user != nil {
			for i, project := range user.Projects {
				if project.Cluster == quota.ClusterName && project.Namespace == quota.Namespace {
					user.Projects = append(user.Projects[:i], user.Projects[i+1:]...)
					break
				}
			}
			authorizer.UpdateUser(user)
		}
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

	cluster := m.clusters.GetClusterByName(clusterInfo.ClusterName)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s doesn't exist", clusterInfo.ClusterName))
	}

	userQuotaID := ctx.Object.(*types.UserQuota).GetID()
	tx, err := BeginTableTransaction(m.clusters.GetDB(), UserQuotaTable)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user quota %s failed %s", userQuotaID, err.Error()))
	}

	defer tx.Rollback()
	quota, err := getUserQuotaFromDB(tx, userQuotaID)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user quota %s failed %s", userQuotaID, err.Error()))
	}

	if quota.Status != types.StatusUserQuotaProcessing {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user quota %s failed: only approval request that status is processing", userQuotaID))
	}

	var oldK8sResourceQuota *corev1.ResourceQuota
	resourceQuota := &types.ResourceQuota{
		Name: quota.Namespace,
		Limits: map[string]string{
			"requests.storage": quota.Storage,
		},
	}

	exists, err := hasNamespace(cluster.KubeClient, quota.Namespace)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("check user %s namespace %s whether exists failed %s",
				quota.UserName, quota.Namespace, err.Error()))
	}

	if exists == false {
		if err := createNamespace(cluster.KubeClient, quota.Namespace); err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("create user %s namespace %s failed %s",
					quota.UserName, quota.Namespace, err.Error()))
		}

		if err := createResourceQuota(cluster.KubeClient, quota.Namespace, resourceQuota); err != nil {
			deleteNamespace(cluster.KubeClient, quota.Namespace)
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("create user %s resourcequota with namespace %s failed %s",
					quota.UserName, quota.Namespace, err.Error()))
		}
	} else {
		oldK8sResourceQuota, err = updateResourceQuota(cluster.KubeClient, quota.Namespace, resourceQuota.Limits)
		if err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("update user %s resourcequota with namespace %s failed %s",
					quota.UserName, quota.Namespace, err.Error()))
		}
	}

	setUserQuotaByAdmin(quota, clusterInfo.ClusterName, "", types.StatusUserQuotaApproval)
	value, err := json.Marshal(quota)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("marshal user quota to storage value failed: %s", err.Error()))
	}

	rollbackResource := func() {
		if exists == false {
			deleteNamespace(cluster.KubeClient, quota.Namespace)
		} else {
			cluster.KubeClient.Update(context.TODO(), oldK8sResourceQuota)
		}
	}

	if err := tx.Update(userQuotaID, value); err != nil {
		rollbackResource()
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user %s quota with namespace %s failed %s",
				quota.UserName, quota.Namespace, err.Error()))
	}

	if err := tx.Commit(); err != nil {
		rollbackResource()
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("approval user %s quota with namespace %s failed %s",
				quota.UserName, quota.Namespace, err.Error()))
	}

	if exists == false {
		authorizer := m.clusters.GetAuthorizer()
		user := authorizer.GetUser(quota.UserName)
		if user != nil {
			user.Projects = append(user.Projects, types.Project{
				Cluster:   clusterInfo.ClusterName,
				Namespace: quota.Namespace,
			})
			authorizer.UpdateUser(user)
		}
	}

	return nil
}

func (m *UserQuotaManager) reject(ctx *resttypes.Context) *resttypes.APIError {
	rejection, ok := ctx.Action.Input.(*types.Rejection)
	if ok == false || rejection.Reason == "" {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "rejection param is not valid")
	}

	userQuotaID := ctx.Object.(*types.UserQuota).GetID()
	tx, err := BeginTableTransaction(m.clusters.GetDB(), UserQuotaTable)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user quota %s failed %s", userQuotaID, err.Error()))
	}

	defer tx.Rollback()
	quota, err := getUserQuotaFromDB(tx, userQuotaID)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user quota %s failed: %s", userQuotaID, err.Error()))
	}

	if quota.Status != types.StatusUserQuotaProcessing {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user quota %s failed: only reject request that status is processing", userQuotaID))
	}

	setUserQuotaByAdmin(quota, quota.ClusterName, rejection.Reason, types.StatusUserQuotaRejection)
	value, err := json.Marshal(quota)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("marshal user quota to storage value failed: %s", err.Error()))
	}

	if err := tx.Update(userQuotaID, value); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user %s quota with namespace %s failed %s",
				quota.UserName, quota.Namespace, err.Error()))
	}

	if err := tx.Commit(); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("reject user %s quota with namespace %s failed %s",
				quota.UserName, quota.Namespace, err.Error()))
	}
	return nil
}

func setUserQuota(userQuota *types.UserQuota, userName, requestType string, creationTimestamp time.Time) {
	userQuota.Name = userQuota.Namespace
	userQuota.SetID(userQuota.Name)
	userQuota.SetType(types.UserQuotaType)
	userQuota.SetCreationTimestamp(creationTimestamp)
	userQuota.Status = types.StatusUserQuotaProcessing
	userQuota.UserName = userName
	userQuota.RequestType = requestType
}

func setUserQuotaByAdmin(userQuota *types.UserQuota, clusterName, reason, status string) {
	userQuota.ClusterName = clusterName
	userQuota.RejectionReason = reason
	userQuota.Status = status
	userQuota.ResponseTimestamp = resttypes.ISOTime(time.Now())
}

func updateResourceQuota(cli client.Client, namespace string, limits map[string]string) (*corev1.ResourceQuota, error) {
	k8sResourceQuota, err := getResourceQuota(cli, namespace, namespace)
	if err != nil {
		return nil, err
	}

	k8sHard, err := scQuotaResourceListToK8sResourceList(limits)
	if err != nil {
		return nil, err
	}

	oldHard := k8sResourceQuota.Spec.Hard
	k8sResourceQuota.Spec.Hard = k8sHard
	if err := cli.Update(context.TODO(), k8sResourceQuota); err != nil {
		return nil, err
	}

	k8sResourceQuota.ResourceVersion = ""
	k8sResourceQuota.Spec.Hard = oldHard
	return k8sResourceQuota, nil
}

func storageResourceValueToSCUserQuota(value []byte) (*types.UserQuota, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("value from db should not be empty")
	}

	var userQuota types.UserQuota
	if err := json.Unmarshal(value, &userQuota); err != nil {
		return nil, err
	}

	userQuota.SetType(types.UserQuotaType)
	return &userQuota, nil
}

func BeginTableTransaction(db storage.DB, tableName string) (storage.Transaction, error) {
	table, err := db.CreateOrGetTable(tableName)
	if err != nil {
		return nil, fmt.Errorf("get table failed: %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	return tx, nil
}

func getUserQuotaFromDB(tx storage.Transaction, id string) (*types.UserQuota, error) {
	value, err := tx.Get(id)
	if err != nil {
		return nil, err
	}

	return storageResourceValueToSCUserQuota(value)
}

func IsExistsNamespaceInDB(db storage.DB, tableName, namespace string) (bool, error) {
	tx, err := BeginTableTransaction(db, tableName)
	if err != nil {
		return false, err
	}

	value, _ := tx.Get(namespace)
	tx.Commit()
	return value != nil, nil
}

var namespaceRegex = regexp.MustCompile("[-a-z0-9]")

func checkUserQuotaParamsValid(quota *types.UserQuota) error {
	if len(namespaceRegex.FindAllString(quota.Namespace, -1)) != len(quota.Namespace) {
		return fmt.Errorf("namespace %s is invalid, must match regex [-a-z0-9]", quota.Namespace)
	}

	if _, err := resource.ParseQuantity(quota.CPU); err != nil {
		return fmt.Errorf("cpu %s is invalid: %s", quota.CPU, err.Error())
	}

	if _, err := resource.ParseQuantity(quota.Memory); err != nil {
		return fmt.Errorf("memory %s is invalid: %s", quota.Memory, err.Error())
	}

	if _, err := resource.ParseQuantity(quota.Storage); err != nil {
		return fmt.Errorf("storage %s is invalid: %s", quota.Storage, err.Error())
	}

	return nil
}

package handler

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	ErrDuplicateKeyInConfigMap = errors.New("duplicate key in configmap")
	ErrUpdateDeletingConfigMap = errors.New("configmap is deleting")
)

type ConfigMapManager struct {
	clusters *ClusterManager
}

func newConfigMapManager(clusters *ClusterManager) *ConfigMapManager {
	return &ConfigMapManager{clusters: clusters}
}

func (m *ConfigMapManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	cm := ctx.Resource.(*types.ConfigMap)
	err := createConfigMap(cluster.GetKubeClient(), namespace, cm)
	if err == nil {
		cm.SetID(cm.Name)
		return cm, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate configmap name %s", cm.Name))
	} else if err == ErrDuplicateKeyInConfigMap {
		return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate key in configmap %s", cm.Name))
	} else {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create configmap failed %s", err.Error()))
	}
}

func (m *ConfigMapManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	cm := ctx.Resource.(*types.ConfigMap)
	if err := updateConfigMap(cluster.GetKubeClient(), namespace, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found configmap %s", cm.GetID()))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update configmap failed %s", err.Error()))
	}
	return cm, nil
}

func (m *ConfigMapManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sConfigMaps, err := getConfigMaps(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found configmaps")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list configmaps failed %s", err.Error()))
	}

	var cms []*types.ConfigMap
	for _, cm := range k8sConfigMaps.Items {
		cms = append(cms, k8sConfigMapToSCConfigMap(&cm))
	}
	return cms, nil
}

func (m ConfigMapManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	cm := ctx.Resource.(*types.ConfigMap)
	k8sConfigMap, err := getConfigMap(cluster.GetKubeClient(), namespace, cm.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found configmap %s", cm.GetID()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get configmap %s failed %s", cm.GetID(), err.Error()))
	}

	return k8sConfigMapToSCConfigMap(k8sConfigMap), nil
}

func (m ConfigMapManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	cm := ctx.Resource.(*types.ConfigMap)
	err := deleteConfigMap(cluster.GetKubeClient(), namespace, cm.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("configmap %s desn't exist", namespace))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete configmap failed %s", err.Error()))
	}
	return nil
}

func getConfigMap(cli client.Client, namespace, name string) (*corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &cm)
	return &cm, err
}

func getConfigMaps(cli client.Client, namespace string) (*corev1.ConfigMapList, error) {
	cms := corev1.ConfigMapList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &cms)
	return &cms, err
}

func createConfigMap(cli client.Client, namespace string, cm *types.ConfigMap) error {
	k8sConfigMap, err := scConfigMapToK8sConfigMap(cm, namespace)
	if err != nil {
		return err
	} else {
		return cli.Create(context.TODO(), k8sConfigMap)
	}
}

func updateConfigMap(cli client.Client, namespace string, cm *types.ConfigMap) error {
	target, err := getConfigMap(cli, namespace, cm.GetID())
	if err != nil {
		return err
	}

	if target.GetDeletionTimestamp() != nil {
		return ErrUpdateDeletingConfigMap
	}

	k8sConfigMap, err := scConfigMapToK8sConfigMap(cm, namespace)
	if err != nil {
		return err
	} else {
		target.Data = k8sConfigMap.Data
		return cli.Update(context.TODO(), target)
	}
}

func scConfigMapToK8sConfigMap(cm *types.ConfigMap, namespace string) (*corev1.ConfigMap, error) {
	data := make(map[string]string)
	for _, c := range cm.Configs {
		if _, ok := data[c.Name]; ok {
			return nil, ErrDuplicateKeyInConfigMap
		}
		data[c.Name] = c.Data
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cm.Name, Namespace: namespace},
		Data:       data,
	}, nil
}

func deleteConfigMap(cli client.Client, namespace, name string) error {
	deploy := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), deploy)
}

func k8sConfigMapToSCConfigMap(k8sConfigMap *corev1.ConfigMap) *types.ConfigMap {
	var configs []types.Config
	for n, d := range k8sConfigMap.Data {
		configs = append(configs, types.Config{
			Name: n,
			Data: d,
		})
	}
	cm := &types.ConfigMap{
		Name:    k8sConfigMap.Name,
		Configs: configs,
	}
	cm.SetID(k8sConfigMap.Name)
	cm.SetCreationTimestamp(k8sConfigMap.CreationTimestamp.Time)
	if k8sConfigMap.GetDeletionTimestamp() != nil {
		cm.SetDeletionTimestamp(k8sConfigMap.DeletionTimestamp.Time)
	}
	return cm
}

package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type ResourceQuotaManager struct {
	clusters *ClusterManager
}

func newResourceQuotaManager(clusters *ClusterManager) *ResourceQuotaManager {
	return &ResourceQuotaManager{clusters: clusters}
}

func (m *ResourceQuotaManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	resourceQuota := ctx.Resource.(*types.ResourceQuota)
	err := createResourceQuota(cluster.GetKubeClient(), namespace, resourceQuota)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate resourceQuota name %s", resourceQuota.Name))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create resourceQuota failed %s", err.Error()))
		}
	}

	resourceQuota.SetID(resourceQuota.Name)
	return resourceQuota, nil
}

func (m *ResourceQuotaManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sResourceQuotas, err := getResourceQuotas(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list resourceQuota info failed:%s", err.Error())
		}
		return nil
	}

	var resourceQuotas []*types.ResourceQuota
	for _, item := range k8sResourceQuotas.Items {
		resourceQuotas = append(resourceQuotas, k8sResourceQuotaToSCResourceQuota(&item))
	}
	return resourceQuotas
}

func (m *ResourceQuotaManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	resourceQuota := ctx.Resource.(*types.ResourceQuota)
	k8sResourceQuota, err := getResourceQuota(cluster.GetKubeClient(), namespace, resourceQuota.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get resourceQuota info failed:%s", err.Error())
		}
		return nil
	}

	return k8sResourceQuotaToSCResourceQuota(k8sResourceQuota)
}

func (m *ResourceQuotaManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	resourceQuota := ctx.Resource.(*types.ResourceQuota)
	if err := deleteResourceQuota(cluster.GetKubeClient(), namespace, resourceQuota.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("resourceQuota %s with namespace %s desn't exist", resourceQuota.GetID(), namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete resourceQuota failed %s", err.Error()))
		}
	}

	return nil
}

func getResourceQuota(cli client.Client, namespace, name string) (*corev1.ResourceQuota, error) {
	resourceQuota := corev1.ResourceQuota{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &resourceQuota)
	return &resourceQuota, err
}

func getResourceQuotas(cli client.Client, namespace string) (*corev1.ResourceQuotaList, error) {
	resourceQuotas := corev1.ResourceQuotaList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &resourceQuotas)
	return &resourceQuotas, err
}

func createResourceQuota(cli client.Client, namespace string, resourceQuota *types.ResourceQuota) error {
	k8sHard, err := scQuotaResourceListToK8sResourceList(resourceQuota.Limits)
	if err != nil {
		return err
	}

	k8sResourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceQuota.Name,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: k8sHard,
		},
	}
	return cli.Create(context.TODO(), k8sResourceQuota)
}

func scQuotaResourceListToK8sResourceList(resourceList map[string]string) (corev1.ResourceList, error) {
	k8sResourceList := make(map[corev1.ResourceName]apiresource.Quantity)
	for name, quantity := range resourceList {
		k8sResourceName, err := scQuotaResourceNameToK8sResourceName(name)
		if err != nil {
			return nil, fmt.Errorf("parse resource name %s failed: %s", name, err.Error())
		}

		k8sQuantity, err := apiresource.ParseQuantity(quantity)
		if err != nil {
			return nil, fmt.Errorf("parse resource %s quantity %s failed: %s", name, quantity, err.Error())
		}

		k8sResourceList[k8sResourceName] = k8sQuantity
	}
	return k8sResourceList, nil
}

func deleteResourceQuota(cli client.Client, namespace, name string) error {
	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), resourceQuota)
}

func k8sResourceQuotaToSCResourceQuota(k8sResourceQuota *corev1.ResourceQuota) *types.ResourceQuota {
	resourceQuota := &types.ResourceQuota{
		Name:   k8sResourceQuota.ObjectMeta.Name,
		Limits: k8sResourceListToSCQuotaResourceList(k8sResourceQuota.Spec.Hard),
		Status: types.ResourceQuotaStatus{
			Limits: k8sResourceListToSCQuotaResourceList(k8sResourceQuota.Status.Hard),
			Used:   k8sResourceListToSCQuotaResourceList(k8sResourceQuota.Status.Used),
		},
	}
	resourceQuota.SetID(k8sResourceQuota.Name)
	resourceQuota.SetCreationTimestamp(k8sResourceQuota.CreationTimestamp.Time)
	if k8sResourceQuota.GetDeletionTimestamp() != nil {
		resourceQuota.SetDeletionTimestamp(k8sResourceQuota.DeletionTimestamp.Time)
	}
	return resourceQuota
}

func k8sResourceListToSCQuotaResourceList(k8sResourceList corev1.ResourceList) map[string]string {
	resourceList := make(map[string]string)
	for name, quantity := range k8sResourceList {
		if name == corev1.ResourceRequestsCPU || name == corev1.ResourceRequestsMemory ||
			name == corev1.ResourceLimitsCPU || name == corev1.ResourceLimitsMemory ||
			name == corev1.ResourceRequestsStorage {
			resourceList[string(name)] = quantity.String()
		}
	}
	return resourceList
}

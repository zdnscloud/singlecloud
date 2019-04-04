package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type LimitRangeManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newLimitRangeManager(clusters *ClusterManager) *LimitRangeManager {
	return &LimitRangeManager{clusters: clusters}
}

func (m *LimitRangeManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	limitRange := ctx.Object.(*types.LimitRange)
	err := createLimitRange(cluster.KubeClient, namespace, limitRange)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate limitRange name %s", limitRange.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create limitRange failed %s", err.Error()))
		}
	}

	limitRange.SetID(limitRange.Name)
	return limitRange, nil
}

func (m *LimitRangeManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sLimitRanges, err := getLimitRanges(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list limitRange info failed:%s", err.Error())
		}
		return nil
	}

	var limitRanges []*types.LimitRange
	for _, item := range k8sLimitRanges.Items {
		limitRanges = append(limitRanges, k8sLimitRangeToSCLimitRange(&item))
	}
	return limitRanges
}

func (m *LimitRangeManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	limitRange := ctx.Object.(*types.LimitRange)
	k8sLimitRange, err := getLimitRange(cluster.KubeClient, namespace, limitRange.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get limitRange info failed:%s", err.Error())
		}
		return nil
	}

	return k8sLimitRangeToSCLimitRange(k8sLimitRange)
}

func (m *LimitRangeManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	limitRange := ctx.Object.(*types.LimitRange)
	if err := deleteLimitRange(cluster.KubeClient, namespace, limitRange.GetID()); err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("limitRange %s with namespace %s desn't exist", limitRange.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete limitRange failed %s", err.Error()))
		}
	}

	return nil
}

func getLimitRange(cli client.Client, namespace, name string) (*corev1.LimitRange, error) {
	limitRange := corev1.LimitRange{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &limitRange)
	return &limitRange, err
}

func getLimitRanges(cli client.Client, namespace string) (*corev1.LimitRangeList, error) {
	limitRanges := corev1.LimitRangeList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &limitRanges)
	return &limitRanges, err
}

func createLimitRange(cli client.Client, namespace string, limitRange *types.LimitRange) error {
	var k8sLimitRangeItems []corev1.LimitRangeItem
	for _, limit := range limitRange.Limits {
		k8sLimitRangeType, err := scLimitRangeTypeToK8sLimitRangeType(limit.Type)
		if err != nil {
			return err
		}

		max, err := scLimitResourceListToK8sResourceList(limit.Max)
		if err != nil {
			return err
		}

		min, err := scLimitResourceListToK8sResourceList(limit.Min)
		if err != nil {
			return err
		}

		defaults, err := scLimitResourceListToK8sResourceList(limit.Default)
		if err != nil {
			return err
		}

		defaultRequest, err := scLimitResourceListToK8sResourceList(limit.DefaultRequest)
		if err != nil {
			return err
		}

		maxLimitRequestRatio, err := scLimitResourceListToK8sResourceList(limit.MaxLimitRequestRatio)
		if err != nil {
			return err
		}

		k8sLimitRangeItems = append(k8sLimitRangeItems, corev1.LimitRangeItem{
			Type:                 k8sLimitRangeType,
			Max:                  max,
			Min:                  min,
			Default:              defaults,
			DefaultRequest:       defaultRequest,
			MaxLimitRequestRatio: maxLimitRequestRatio,
		})
	}

	k8sLimitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitRange.Name,
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: k8sLimitRangeItems,
		},
	}
	return cli.Create(context.TODO(), k8sLimitRange)
}

func scLimitResourceListToK8sResourceList(resourceList map[string]string) (corev1.ResourceList, error) {
	k8sResourceList := make(map[corev1.ResourceName]resource.Quantity)
	for name, quantity := range resourceList {
		k8sResourceName, err := scLimitsResourceNameToK8sResourceName(name)
		if err != nil {
			return nil, fmt.Errorf("parse resource name %s failed: %s", name, err.Error())
		}

		k8sQuantity, err := resource.ParseQuantity(quantity)
		if err != nil {
			return nil, fmt.Errorf("parse resource %s quantity %s failed: %s", name, quantity, err.Error())
		}

		k8sResourceList[k8sResourceName] = k8sQuantity
	}
	return k8sResourceList, nil
}

func deleteLimitRange(cli client.Client, namespace, name string) error {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), limitRange)
}

func k8sLimitRangeToSCLimitRange(k8sLimitRange *corev1.LimitRange) *types.LimitRange {
	var limitRangeItems []types.LimitRangeItem
	for _, limit := range k8sLimitRange.Spec.Limits {
		limitRangeItems = append(limitRangeItems, types.LimitRangeItem{
			Type:                 string(limit.Type),
			Max:                  k8sResourceListToSCResourceList(limit.Max),
			Min:                  k8sResourceListToSCResourceList(limit.Min),
			Default:              k8sResourceListToSCResourceList(limit.Default),
			DefaultRequest:       k8sResourceListToSCResourceList(limit.DefaultRequest),
			MaxLimitRequestRatio: k8sResourceListToSCResourceList(limit.MaxLimitRequestRatio),
		})
	}

	limitRange := &types.LimitRange{
		Name:   k8sLimitRange.ObjectMeta.Name,
		Limits: limitRangeItems,
	}
	limitRange.SetID(k8sLimitRange.Name)
	limitRange.SetType(types.LimitRangeType)
	limitRange.SetCreationTimestamp(k8sLimitRange.CreationTimestamp.Time)
	return limitRange
}

func k8sResourceListToSCResourceList(k8sResourceList corev1.ResourceList) map[string]string {
	resourceList := make(map[string]string)
	for name, quantity := range k8sResourceList {
		resourceList[string(name)] = quantity.String()
	}
	return resourceList
}

package handler

import (
	"context"
	"fmt"

	asv1 "k8s.io/api/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const workloadAPIVersion = "apps/v1"

type HorizontalPodAutoscalerManager struct {
	clusters *ClusterManager
}

func newHorizontalPodAutoscalerManager(clusters *ClusterManager) *HorizontalPodAutoscalerManager {
	return &HorizontalPodAutoscalerManager{clusters: clusters}
}

func (m *HorizontalPodAutoscalerManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	hpa := ctx.Resource.(*types.HorizontalPodAutoscaler)
	if err := createHorizontalPodAutoscaler(cluster.KubeClient, namespace, hpa); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource,
				fmt.Sprintf("duplicate horizontalpodautoscaler name %s", hpa.Name))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("create horizontalpodautoscaler failed %s", err.Error()))
		}
	}

	hpa.SetID(hpa.Name)
	return hpa, nil
}

func createHorizontalPodAutoscaler(cli client.Client, namespace string, hpa *types.HorizontalPodAutoscaler) error {
	minReplicas := int32(hpa.MinReplicas)
	cpuUtilizationPercentage := int32(hpa.CPUUtilizationPercentage)
	k8sHpa := &asv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hpa.Name,
			Namespace: namespace,
		},
		Spec: asv1.HorizontalPodAutoscalerSpec{
			MinReplicas:                    &minReplicas,
			MaxReplicas:                    int32(hpa.MaxReplicas),
			TargetCPUUtilizationPercentage: &cpuUtilizationPercentage,
			ScaleTargetRef: asv1.CrossVersionObjectReference{
				APIVersion: workloadAPIVersion,
				Kind:       hpa.ScaleTargetKind,
				Name:       hpa.ScaleTargetName,
			},
		},
	}

	return cli.Create(context.TODO(), k8sHpa)
}

func (m *HorizontalPodAutoscalerManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sHpas, err := getHorizontalPodAutoscalers(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list horizontalpodautoscaler info failed:%s", err.Error())
		}
		return nil
	}

	var hpas []*types.HorizontalPodAutoscaler
	for _, item := range k8sHpas.Items {
		hpas = append(hpas, k8sHpaToScHpa(&item))
	}

	return hpas
}

func getHorizontalPodAutoscalers(cli client.Client, namespace string) (*asv1.HorizontalPodAutoscalerList, error) {
	hpas := asv1.HorizontalPodAutoscalerList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &hpas)
	return &hpas, err
}

func k8sHpaToScHpa(k8sHpa *asv1.HorizontalPodAutoscaler) *types.HorizontalPodAutoscaler {
	var minReplicas int
	if k8sHpa.Spec.MinReplicas != nil {
		minReplicas = int(*k8sHpa.Spec.MinReplicas)
	}

	var cpuUtilizationPercentage int
	if k8sHpa.Spec.TargetCPUUtilizationPercentage != nil {
		cpuUtilizationPercentage = int(*k8sHpa.Spec.TargetCPUUtilizationPercentage)
	}

	var currentCPUUtilizationPercentage int
	if k8sHpa.Status.CurrentCPUUtilizationPercentage != nil {
		currentCPUUtilizationPercentage = int(*k8sHpa.Status.CurrentCPUUtilizationPercentage)
	}

	return &types.HorizontalPodAutoscaler{
		Name:                     k8sHpa.Name,
		ScaleTargetKind:          k8sHpa.Spec.ScaleTargetRef.Kind,
		ScaleTargetName:          k8sHpa.Spec.ScaleTargetRef.Name,
		MaxReplicas:              int(k8sHpa.Spec.MaxReplicas),
		MinReplicas:              minReplicas,
		CPUUtilizationPercentage: cpuUtilizationPercentage,
		Status: types.HorizontalPodAutoscalerStatus{
			CurrentReplicas:                 int(k8sHpa.Status.CurrentReplicas),
			DesiredReplicas:                 int(k8sHpa.Status.DesiredReplicas),
			CurrentCPUUtilizationPercentage: currentCPUUtilizationPercentage,
		},
	}
}

func (m *HorizontalPodAutoscalerManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	hpa := ctx.Resource.(*types.HorizontalPodAutoscaler)
	k8sHpa, err := getHorizontalPodAutoscaler(cluster.KubeClient, namespace, hpa.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get horizontalpodautoscaler info failed:%s", err.Error())
		}
		return nil
	}

	return k8sHpaToScHpa(k8sHpa)
}

func getHorizontalPodAutoscaler(cli client.Client, namespace, name string) (*asv1.HorizontalPodAutoscaler, error) {
	hpa := asv1.HorizontalPodAutoscaler{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &hpa)
	return &hpa, err
}

func (m *HorizontalPodAutoscalerManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	hpa := ctx.Resource.(*types.HorizontalPodAutoscaler)
	k8sHpa, err := getHorizontalPodAutoscaler(cluster.KubeClient, namespace, hpa.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("horizontalpodautoscaler %s with namespace %s doesn't exist", hpa.GetID(), namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("get horizontalpodautoscaler failed %s", err.Error()))
		}
	}

	resetHorizontalPodAutoscaler(k8sHpa, hpa)
	if err := cluster.KubeClient.Update(context.TODO(), k8sHpa); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update horizontalpodautoscaler failed %s", err.Error()))
	}

	return hpa, nil
}

func resetHorizontalPodAutoscaler(k8sHpa *asv1.HorizontalPodAutoscaler, hpa *types.HorizontalPodAutoscaler) {
	minReplicas := int32(hpa.MinReplicas)
	cpuUtilizationPercentage := int32(hpa.CPUUtilizationPercentage)
	k8sHpa.Spec.MinReplicas = &minReplicas
	k8sHpa.Spec.MaxReplicas = int32(hpa.MaxReplicas)
	k8sHpa.Spec.TargetCPUUtilizationPercentage = &cpuUtilizationPercentage
}

func (m *HorizontalPodAutoscalerManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	hpa := ctx.Resource.(*types.HorizontalPodAutoscaler)
	if err := deleteHorizontalPodAutoscaler(cluster.KubeClient, namespace, hpa.GetID()); err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("horizontalpodautoscaler %s with namespace %s doesn't exist", hpa.GetID(), namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete horizontalpodautoscaler failed %s", err.Error()))
		}
	}

	return nil
}

func deleteHorizontalPodAutoscaler(cli client.Client, namespace, name string) error {
	hpa := &asv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), hpa)
}

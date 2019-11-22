package handler

import (
	"context"
	"fmt"
	"strconv"

	asv2beta2 "k8s.io/api/autoscaling/v2beta2"
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
	if err := createHPA(cluster.KubeClient, namespace, hpa); err != nil {
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

func createHPA(cli client.Client, namespace string, hpa *types.HorizontalPodAutoscaler) error {
	k8sHpaSpec, err := scHPAToK8sHPASpec(hpa)
	if err != nil {
		return err
	}

	k8sHpa := &asv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hpa.Name,
			Namespace: namespace,
		},
		Spec: k8sHpaSpec,
	}

	return cli.Create(context.TODO(), k8sHpa)
}

func scHPAToK8sHPASpec(hpa *types.HorizontalPodAutoscaler) (asv2beta2.HorizontalPodAutoscalerSpec, error) {
	var metrics []asv2beta2.MetricSpec
	for _, metric := range hpa.ResourceMetrics {
		metricSpec, err := scResourceMetricSpecToK8sMetricSpec(metric)
		if err != nil {
			return asv2beta2.HorizontalPodAutoscalerSpec{}, err
		}

		metrics = append(metrics, metricSpec)
	}

	for _, metric := range hpa.CustomMetrics {
		metricSpec, err := scCustomMetricSpecToK8sMetricSpec(metric)
		if err != nil {
			return asv2beta2.HorizontalPodAutoscalerSpec{}, err
		}
		metrics = append(metrics, metricSpec)
	}

	minReplicas := int32(hpa.MinReplicas)
	return asv2beta2.HorizontalPodAutoscalerSpec{
		MinReplicas: &minReplicas,
		MaxReplicas: int32(hpa.MaxReplicas),
		ScaleTargetRef: asv2beta2.CrossVersionObjectReference{
			APIVersion: workloadAPIVersion,
			Kind:       string(hpa.ScaleTargetKind),
			Name:       hpa.ScaleTargetName,
		},
		Metrics: metrics,
	}, nil
}

func scResourceMetricSpecToK8sMetricSpec(metric types.ResourceMetricSpec) (asv2beta2.MetricSpec, error) {
	target, err := scMetricValueToK8sMetricTarget(metric.TargetType, metric.AverageValue, metric.AverageUtilization)
	if err != nil {
		return asv2beta2.MetricSpec{}, err
	}

	name, err := scResourceNameToK8sResourceName(string(metric.ResourceName))
	if err != nil {
		return asv2beta2.MetricSpec{}, err
	}

	return asv2beta2.MetricSpec{
		Type: asv2beta2.ResourceMetricSourceType,
		Resource: &asv2beta2.ResourceMetricSource{
			Name:   name,
			Target: target,
		},
	}, nil
}

func scCustomMetricSpecToK8sMetricSpec(metric types.CustomMetricSpec) (asv2beta2.MetricSpec, error) {
	target, err := scMetricValueToK8sMetricTarget(types.MetricTargetTypeAverageValue, metric.AverageValue, 0)
	if err != nil {
		return asv2beta2.MetricSpec{}, err
	}

	return asv2beta2.MetricSpec{
		Type: asv2beta2.PodsMetricSourceType,
		Pods: &asv2beta2.PodsMetricSource{
			Metric: asv2beta2.MetricIdentifier{
				Name: metric.MetricName,
			},
			Target: target,
		},
	}, nil
}

func scMetricValueToK8sMetricTarget(typ types.MetricTargetType, value string, utilization int) (asv2beta2.MetricTarget, error) {
	switch typ {
	case types.MetricTargetTypeUtilization:
		if utilization == 0 {
			return asv2beta2.MetricTarget{}, fmt.Errorf("averageUtilization must not be empty when type is %s", typ)
		}

		averageUtilization := int32(utilization)
		return asv2beta2.MetricTarget{
			Type:               asv2beta2.UtilizationMetricType,
			AverageUtilization: &averageUtilization,
		}, nil
	case types.MetricTargetTypeAverageValue:
		if value == "" {
			return asv2beta2.MetricTarget{}, fmt.Errorf("averageValue must not be empty when type is %s", typ)
		}

		averageValue, err := apiresource.ParseQuantity(value)
		if err != nil {
			return asv2beta2.MetricTarget{}, fmt.Errorf("parse metric averageValue failed: %s", err.Error())
		}

		return asv2beta2.MetricTarget{
			Type:         asv2beta2.AverageValueMetricType,
			AverageValue: &averageValue,
		}, nil
	default:
		return asv2beta2.MetricTarget{}, fmt.Errorf("metric target type %s is unsupported", typ)
	}
}

func (m *HorizontalPodAutoscalerManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sHpas, err := getHPAs(cluster.KubeClient, namespace)
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

func getHPAs(cli client.Client, namespace string) (*asv2beta2.HorizontalPodAutoscalerList, error) {
	hpas := asv2beta2.HorizontalPodAutoscalerList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &hpas)
	return &hpas, err
}

func k8sHpaToScHpa(k8sHpa *asv2beta2.HorizontalPodAutoscaler) *types.HorizontalPodAutoscaler {
	var minReplicas int
	if k8sHpa.Spec.MinReplicas != nil {
		minReplicas = int(*k8sHpa.Spec.MinReplicas)
	}

	var resourceMetrics []types.ResourceMetricSpec
	var customMetrics []types.CustomMetricSpec
	for _, k8sMetric := range k8sHpa.Spec.Metrics {
		if k8sMetric.Type == asv2beta2.ObjectMetricSourceType {
			continue
		}

		if k8sMetric.Type == asv2beta2.ResourceMetricSourceType && k8sMetric.Resource != nil {
			resourceMetricSpec := k8sMetricSpecToScResourceMetricSpec(k8sMetric.Resource.Name, k8sMetric.Resource.Target.AverageValue,
				k8sMetric.Resource.Target.AverageUtilization)
			resourceMetricSpec.TargetType = types.MetricTargetType(k8sMetric.Resource.Target.Type)
			resourceMetrics = append(resourceMetrics, resourceMetricSpec)
		} else if k8sMetric.Type == asv2beta2.PodsMetricSourceType && k8sMetric.Pods != nil {
			customMetrics = append(customMetrics, k8sMetricSpecToScCustomMetricSpec(k8sMetric.Pods.Metric.Name,
				k8sMetric.Pods.Target.AverageValue))
		}
	}

	var currentMetrics types.MetricStatus
	for _, k8sCurrent := range k8sHpa.Status.CurrentMetrics {
		if k8sCurrent.Type == asv2beta2.ObjectMetricSourceType {
			continue
		}

		if k8sCurrent.Type == asv2beta2.ResourceMetricSourceType && k8sCurrent.Resource != nil {
			currentMetrics.ResourceMetrics = append(currentMetrics.ResourceMetrics, k8sMetricSpecToScResourceMetricSpec(
				k8sCurrent.Resource.Name, k8sCurrent.Resource.Current.AverageValue, k8sCurrent.Resource.Current.AverageUtilization))
		} else if k8sCurrent.Type == asv2beta2.PodsMetricSourceType && k8sCurrent.Pods != nil {
			currentMetrics.CustomMetrics = append(currentMetrics.CustomMetrics, k8sMetricSpecToScCustomMetricSpec(
				k8sCurrent.Pods.Metric.Name, k8sCurrent.Pods.Current.AverageValue))
		}
	}

	hpa := &types.HorizontalPodAutoscaler{
		Name:            k8sHpa.Name,
		ScaleTargetKind: types.ScaleTargetKind(k8sHpa.Spec.ScaleTargetRef.Kind),
		ScaleTargetName: k8sHpa.Spec.ScaleTargetRef.Name,
		MaxReplicas:     int(k8sHpa.Spec.MaxReplicas),
		MinReplicas:     minReplicas,
		ResourceMetrics: resourceMetrics,
		CustomMetrics:   customMetrics,
		Status: types.HorizontalPodAutoscalerStatus{
			CurrentReplicas: int(k8sHpa.Status.CurrentReplicas),
			DesiredReplicas: int(k8sHpa.Status.DesiredReplicas),
			CurrentMetrics:  currentMetrics,
		},
	}
	hpa.SetID(k8sHpa.Name)
	hpa.SetCreationTimestamp(k8sHpa.CreationTimestamp.Time)
	return hpa
}

func k8sMetricSpecToScResourceMetricSpec(k8sResourceName corev1.ResourceName, k8sAverageValue *apiresource.Quantity, k8sAverageUtilization *int32) types.ResourceMetricSpec {
	var averageUtilization int
	if k8sAverageUtilization != nil {
		averageUtilization = int(*k8sAverageUtilization)
	}

	var averageValue string
	if k8sAverageValue != nil {
		switch k8sResourceName {
		case corev1.ResourceCPU:
			averageValue = strconv.Itoa(int(k8sAverageValue.MilliValue()))
		case corev1.ResourceMemory:
			averageValue = strconv.Itoa(int(k8sAverageValue.Value()))
		}
	}

	return types.ResourceMetricSpec{
		ResourceName:       types.ResourceName(k8sResourceName),
		AverageValue:       averageValue,
		AverageUtilization: averageUtilization,
	}
}

func k8sMetricSpecToScCustomMetricSpec(metricName string, k8sAverageValue *apiresource.Quantity) types.CustomMetricSpec {
	var averageValue string
	if k8sAverageValue != nil {
		averageValue = strconv.Itoa(int(k8sAverageValue.Value()))
	}

	return types.CustomMetricSpec{
		MetricName:   metricName,
		AverageValue: averageValue,
	}
}

func (m *HorizontalPodAutoscalerManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	hpa := ctx.Resource.(*types.HorizontalPodAutoscaler)
	k8sHpa, err := getHPA(cluster.KubeClient, namespace, hpa.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get horizontalpodautoscaler info failed:%s", err.Error())
		}
		return nil
	}

	return k8sHpaToScHpa(k8sHpa)
}

func getHPA(cli client.Client, namespace, name string) (*asv2beta2.HorizontalPodAutoscaler, error) {
	hpa := asv2beta2.HorizontalPodAutoscaler{}
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
	k8sHpa, err := getHPA(cluster.KubeClient, namespace, hpa.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("horizontalpodautoscaler %s with namespace %s doesn't exist", hpa.GetID(), namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("get horizontalpodautoscaler failed %s", err.Error()))
		}
	}

	if err := updateHPA(cluster.KubeClient, k8sHpa, hpa); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update horizontalpodautoscaler failed %s", err.Error()))
	}

	return hpa, nil
}

func updateHPA(cli client.Client, k8sHpa *asv2beta2.HorizontalPodAutoscaler, hpa *types.HorizontalPodAutoscaler) error {
	k8sHpaSpec, err := scHPAToK8sHPASpec(hpa)
	if err != nil {
		return err
	}

	k8sHpa.Spec = k8sHpaSpec
	return cli.Update(context.TODO(), k8sHpa)
}

func (m *HorizontalPodAutoscalerManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	hpa := ctx.Resource.(*types.HorizontalPodAutoscaler)
	if err := deleteHPA(cluster.KubeClient, namespace, hpa.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("horizontalpodautoscaler %s with namespace %s doesn't exist", hpa.GetID(), namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete horizontalpodautoscaler failed %s", err.Error()))
		}
	}

	return nil
}

func deleteHPA(cli client.Client, namespace, name string) error {
	hpa := &asv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), hpa)
}

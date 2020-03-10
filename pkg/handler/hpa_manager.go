package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	asv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	WorkloadAPIVersion                = "apps/v1"
	PrometheusAdapterNamespace        = "zcloud"
	PrometheusAdapter                 = "monitor-prometheus-adapter"
	PrometheusAdapterConfigMapDataKey = "config.yaml"

	SeriesQueryTemplate   = "{__name__=~\"%s\",kubernetes_pod_name!=\"\",kubernetes_namespace=\"%s\"}"
	NameMatchesTemplate   = "^%s$"
	NameAsTemplate        = "%s_%s_%s_%s"
	MetricsQueryTemplate  = "sum(%s{%s}) by (<<.GroupBy>>)"
	LabelMatchersTemplate = "%s=\"%s\","
)

var DefaultRuleResources = RuleResources{
	Overrides: map[string]map[string]string{
		"kubernetes_namespace": map[string]string{
			"resource": "namespace",
		},
		"kubernetes_pod_name": map[string]string{
			"resource": "pod",
		}},
}

type HorizontalPodAutoscalerManager struct {
	clusters        *ClusterManager
	workloadEventCh <-chan interface{}
}

func newHorizontalPodAutoscalerManager(clusters *ClusterManager) *HorizontalPodAutoscalerManager {
	m := &HorizontalPodAutoscalerManager{
		clusters:        clusters,
		workloadEventCh: eb.SubscribeResourceEvent(types.Deployment{}, types.StatefulSet{}),
	}
	go m.eventLoop()
	return m
}

func (m *HorizontalPodAutoscalerManager) eventLoop() {
	for {
		event := <-m.workloadEventCh
		switch e := event.(type) {
		case eb.ResourceDeleteEvent:
			if err := m.deleteHPAWhenDeleteWorkload(e.Resource); err != nil {
				log.Warnf("delete workload %s/%s hpa failed: %s", e.Resource.GetType(), e.Resource.GetID(), err.Error())
			}
		}
	}
}

func (m *HorizontalPodAutoscalerManager) deleteHPAWhenDeleteWorkload(r resource.Resource) error {
	cluster := m.clusters.GetClusterForSubResource(r)
	if cluster == nil {
		return fmt.Errorf("no found cluster")
	}
	namespace := r.GetParent().GetID()
	k8sHpa, err := getHPAByWorkload(cluster.GetKubeClient(), namespace, r.GetType(), r.GetID())
	if err != nil {
		return fmt.Errorf("get hpa with namespace %s failed: %s", namespace, err.Error())
	}

	if k8sHpa == nil {
		return nil
	}

	return updatePrometheusAdapterCMAndDeleteHPA(cluster.GetKubeClient(), k8sHpa)
}

func getHPAByWorkload(cli client.Client, namespace, workloadType, workloadName string) (*asv2beta2.HorizontalPodAutoscaler, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{workloadType: workloadName}})
	if err != nil {
		return nil, err
	}

	k8sHpas, err := getHPAs(cli, namespace, selector)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, err
		}
		return nil, nil
	}

	for _, item := range k8sHpas.Items {
		if item.Spec.ScaleTargetRef.Kind == workloadType && item.Spec.ScaleTargetRef.Name == workloadName {
			return &item, nil
		}
	}

	return nil, nil
}

func (m *HorizontalPodAutoscalerManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	hpa := ctx.Resource.(*types.HorizontalPodAutoscaler)
	if err := createHPA(cluster.GetKubeClient(), namespace, hpa); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource,
				fmt.Sprintf("duplicate horizontalpodautoscaler name %s with namespace %s", hpa.Name, namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("create horizontalpodautoscaler %s with namespace %s failed %s", hpa.Name, namespace, err.Error()))
		}
	}

	hpa.SetID(hpa.Name)
	return hpa, nil
}

func createHPA(cli client.Client, namespace string, hpa *types.HorizontalPodAutoscaler) error {
	k8sHpas, err := getHPAs(cli, namespace, nil)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return fmt.Errorf("list horizontalpodautoscalers failed:%s", err.Error())
		}
	}

	for _, item := range k8sHpas.Items {
		if item.Name == hpa.Name ||
			(item.Spec.ScaleTargetRef.Kind == string(hpa.ScaleTargetKind) &&
				item.Spec.ScaleTargetRef.Name == hpa.ScaleTargetName) {
			return fmt.Errorf("duplicate horizontalpodautoscaler %s for %s/%s",
				item.Name, item.Spec.ScaleTargetRef.Kind, item.Spec.ScaleTargetRef.Name)
		}
	}

	k8sHpaSpec, rules, err := scHPAToK8sHPASpec(namespace, hpa)
	if err != nil {
		return err
	}

	if err := updatePrometheusAdapterConfigMap(cli, nil, rules); err != nil {
		return err
	}

	return cli.Create(context.TODO(), &asv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hpa.Name,
			Namespace: namespace,
			Labels:    map[string]string{string(hpa.ScaleTargetKind): hpa.ScaleTargetName},
		},
		Spec: k8sHpaSpec,
	})
}

func scHPAToK8sHPASpec(namespace string, hpa *types.HorizontalPodAutoscaler) (asv2beta2.HorizontalPodAutoscalerSpec, []Rule, error) {
	var metrics []asv2beta2.MetricSpec
	for _, metric := range hpa.ResourceMetrics {
		metricSpec, err := scResourceMetricSpecToK8sMetricSpec(metric)
		if err != nil {
			return asv2beta2.HorizontalPodAutoscalerSpec{}, nil, err
		}

		metrics = append(metrics, metricSpec)
	}

	var rules []Rule
	for _, metric := range hpa.CustomMetrics {
		rule, metricName, err := genPrometheusAdapterConfigMapRule(namespace, hpa.Name, metric)
		if err != nil {
			return asv2beta2.HorizontalPodAutoscalerSpec{}, nil, err
		}

		metricSpec, err := scCustomMetricSpecToK8sMetricSpec(metricName, metric)
		if err != nil {
			return asv2beta2.HorizontalPodAutoscalerSpec{}, nil, err
		}

		metrics = append(metrics, metricSpec)
		rules = append(rules, rule)
	}

	minReplicas := int32(hpa.MinReplicas)
	return asv2beta2.HorizontalPodAutoscalerSpec{
		MinReplicas: &minReplicas,
		MaxReplicas: int32(hpa.MaxReplicas),
		ScaleTargetRef: asv2beta2.CrossVersionObjectReference{
			APIVersion: WorkloadAPIVersion,
			Kind:       string(hpa.ScaleTargetKind),
			Name:       hpa.ScaleTargetName,
		},
		Metrics: metrics,
	}, rules, nil
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

func genPrometheusAdapterConfigMapRule(namespace, hpaName string, metric types.CustomMetricSpec) (Rule, string, error) {
	labelsHash, err := hashMetricLabel(metric.Labels)
	if err != nil {
		return Rule{}, "", fmt.Errorf("hash custom metric labels failed: %s", err.Error())
	}

	metricName := fmt.Sprintf(NameAsTemplate, metric.MetricName, labelsHash, namespace, hpaName)
	return Rule{
		SeriesQuery: fmt.Sprintf(SeriesQueryTemplate, metric.MetricName, namespace),
		Resources:   DefaultRuleResources,
		Name: RuleName{
			Matches: fmt.Sprintf(NameMatchesTemplate, metric.MetricName),
			As:      metricName,
		},
		MetricsQuery: fmt.Sprintf(MetricsQueryTemplate, metric.MetricName, labelsToString(metric.Labels)),
	}, metricName, nil
}

func hashMetricLabel(labels map[string]string) (string, error) {
	data, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(data))[:12], nil
}

func labelsToString(labels map[string]string) string {
	var b bytes.Buffer
	for name, value := range labels {
		b.WriteString(fmt.Sprintf(LabelMatchersTemplate, name, value))
	}
	return strings.TrimSuffix(b.String(), ",")
}

func scCustomMetricSpecToK8sMetricSpec(metricName string, metric types.CustomMetricSpec) (asv2beta2.MetricSpec, error) {
	target, err := scMetricValueToK8sMetricTarget(types.MetricTargetTypeAverageValue, metric.AverageValue, 0)
	if err != nil {
		return asv2beta2.MetricSpec{}, err
	}

	return asv2beta2.MetricSpec{
		Type: asv2beta2.PodsMetricSourceType,
		Pods: &asv2beta2.PodsMetricSource{
			Metric: asv2beta2.MetricIdentifier{
				Name:     metricName,
				Selector: &metav1.LabelSelector{MatchLabels: metric.Labels},
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

func updatePrometheusAdapterConfigMap(cli client.Client, oldRules, newRules []Rule) error {
	if len(oldRules) == 0 && len(newRules) == 0 {
		return nil
	}

	k8sConfigMap, err := getConfigMap(cli, PrometheusAdapterNamespace, PrometheusAdapter)
	if err != nil {
		return fmt.Errorf("get prometheus-adapter configmap failed: %s", err.Error())
	}

	rulesRaw, ok := k8sConfigMap.Data[PrometheusAdapterConfigMapDataKey]
	if ok == false {
		return fmt.Errorf("no found %s in prometheus-adapter configmap data", PrometheusAdapterConfigMapDataKey)
	}

	var config PrometheusAdapterConfig
	if err := yaml.Unmarshal([]byte(rulesRaw), &config); err != nil {
		return fmt.Errorf("unmarshal prometheus-adapter configmap data rules failed: %s", err.Error())
	}

	for _, oldRule := range oldRules {
		exists := false
		for i, rule := range config.Rules {
			if rule.Name.As == oldRule.Name.As {
				exists = true
				config.Rules = append(config.Rules[:i], config.Rules[i+1:]...)
				break
			}
		}

		if exists == false {
			return fmt.Errorf("no found hpa custom metric alias name %s in prometheus-adapter configmap", oldRule.Name.As)
		}
	}

	for _, newRule := range newRules {
		exists := false
		for _, rule := range config.Rules {
			if rule.Name.As == newRule.Name.As {
				exists = true
				break
			}
		}

		if exists {
			return fmt.Errorf("duplicate hpa custom metric alias name %s in prometheus-adapter configmap", newRule.Name.As)
		}

		config.Rules = append(config.Rules, newRule)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal new prometheus-adapter configmap data failed: %s", err.Error())
	}

	k8sConfigMap.Data[PrometheusAdapterConfigMapDataKey] = string(data)
	if err := cli.Update(context.TODO(), k8sConfigMap); err != nil {
		return fmt.Errorf("update prometheus-adapter configmap failed: %s", err.Error())
	}

	k8sPods, err := getOwnerPods(cli, PrometheusAdapterNamespace, types.ResourceTypeDeployment, PrometheusAdapter)
	if err != nil {
		return fmt.Errorf("get prometheus-adapter pods failed: %s", err.Error())
	}

	for _, pod := range k8sPods.Items {
		if err := deletePod(cli, PrometheusAdapterNamespace, pod.Name); err != nil {
			return fmt.Errorf("restart prometheus-adapter pod %s failed: %s", pod.Name, err.Error())
		}
	}

	return nil
}

func (m *HorizontalPodAutoscalerManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sHpas, err := getHPAs(cluster.GetKubeClient(), namespace, nil)
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

func getHPAs(cli client.Client, namespace string, selector labels.Selector) (*asv2beta2.HorizontalPodAutoscalerList, error) {
	hpas := asv2beta2.HorizontalPodAutoscalerList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	if selector != nil {
		listOptions.LabelSelector = selector
	}

	err := cli.List(context.TODO(), listOptions, &hpas)
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
			customMetrics = append(customMetrics, k8sMetricSpecToScCustomMetricSpec(k8sHpa.Namespace, k8sHpa.Name,
				k8sMetric.Pods.Metric.Name, k8sMetric.Pods.Metric.Selector, k8sMetric.Pods.Target.AverageValue))
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
			currentMetrics.CustomMetrics = append(currentMetrics.CustomMetrics, k8sMetricSpecToScCustomMetricSpec(k8sHpa.Namespace,
				k8sHpa.Name, k8sCurrent.Pods.Metric.Name, k8sCurrent.Pods.Metric.Selector, k8sCurrent.Pods.Current.AverageValue))
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
	if k8sHpa.GetDeletionTimestamp() != nil {
		hpa.SetDeletionTimestamp(k8sHpa.DeletionTimestamp.Time)
	}
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

func k8sMetricSpecToScCustomMetricSpec(namespace, hpaName, metricName string, selector *metav1.LabelSelector, k8sAverageValue *apiresource.Quantity) types.CustomMetricSpec {
	var averageValue string
	if k8sAverageValue != nil {
		averageValue = strconv.Itoa(int(k8sAverageValue.Value()))
	}

	labelsHash, _ := hashMetricLabel(selector.MatchLabels)
	return types.CustomMetricSpec{
		MetricName:   strings.TrimSuffix(metricName, fmt.Sprintf(NameAsTemplate, "", labelsHash, namespace, hpaName)),
		Labels:       selector.MatchLabels,
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
	k8sHpa, err := getHPA(cluster.GetKubeClient(), namespace, hpa.GetID())
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
	if err := updateHPA(cluster.GetKubeClient(), namespace, hpa); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("horizontalpodautoscaler %s with namespace %s doesn't exist", hpa.GetID(), namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("update horizontalpodautoscaler failed %s", err.Error()))
		}
	}

	return hpa, nil
}

func updateHPA(cli client.Client, namespace string, hpa *types.HorizontalPodAutoscaler) error {
	k8sHpa, err := getHPA(cli, namespace, hpa.GetID())
	if err != nil {
		return err
	}

	k8sHpaSpec, newRules, err := scHPAToK8sHPASpec(namespace, hpa)
	if err != nil {
		return err
	}

	if err := updatePrometheusAdapterConfigMap(cli, getOldRules(k8sHpa.Spec.Metrics), newRules); err != nil {
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
	if err := deleteHPA(cluster.GetKubeClient(), namespace, hpa.GetID()); err != nil {
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
	k8sHpa, err := getHPA(cli, namespace, name)
	if err != nil {
		return err
	}

	return updatePrometheusAdapterCMAndDeleteHPA(cli, k8sHpa)
}

func updatePrometheusAdapterCMAndDeleteHPA(cli client.Client, k8sHpa *asv2beta2.HorizontalPodAutoscaler) error {
	if err := updatePrometheusAdapterConfigMap(cli, getOldRules(k8sHpa.Spec.Metrics), nil); err != nil {
		return err
	}

	return cli.Delete(context.TODO(), k8sHpa)
}

func getOldRules(k8sHpaSpecMetrics []asv2beta2.MetricSpec) []Rule {
	var rules []Rule
	for _, k8sMetric := range k8sHpaSpecMetrics {
		if k8sMetric.Type == asv2beta2.PodsMetricSourceType && k8sMetric.Pods != nil {
			rules = append(rules, Rule{
				Name: RuleName{
					As: k8sMetric.Pods.Metric.Name,
				},
			})
		}
	}
	return rules
}

type PrometheusAdapterConfig struct {
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	SeriesQuery   string         `yaml:"seriesQuery"`
	SeriesFilters []SeriesFilter `yaml:"seriesFilters"`
	Resources     RuleResources  `yaml:"resources"`
	Name          RuleName       `yaml:"name"`
	MetricsQuery  string         `yaml:"metricsQuery"`
}

type SeriesFilter struct {
	IsNot string `yaml:"isNot,omitempty"`
	Is    string `yaml:"is,omitempty"`
}

type RuleResources struct {
	Template  string                       `yaml:"template,omitempty"`
	Overrides map[string]map[string]string `yaml:"overrides,omitempty"`
}

type RuleName struct {
	Matches string `yaml:"matches"`
	As      string `yaml:"as"`
}

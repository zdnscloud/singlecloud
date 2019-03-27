package handler

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AnnkeyForDaemonSetAdvancedoption = "zcloud_daemonset_advanded_options"
)

type DaemonSetManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newDaemonSetManager(clusters *ClusterManager) *DaemonSetManager {
	return &DaemonSetManager{clusters: clusters}
}

func (m *DaemonSetManager) Create(obj resttypes.Object, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := obj.GetParent().GetID()
	daemonSet := obj.(*types.DaemonSet)
	if err := createDaemonSet(cluster.KubeClient, namespace, daemonSet); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate daemonSet name %s", daemonSet.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create daemonSet failed %s", err.Error()))
		}
	}

	daemonSet.SetID(daemonSet.Name)
	if err := createServiceAndIngress(daemonSet.AdvancedOptions, cluster.KubeClient, namespace, daemonSet.Name); err != nil {
		deleteDaemonSet(cluster.KubeClient, namespace, daemonSet.Name)
		return nil, err
	}

	return daemonSet, nil
}

func (m *DaemonSetManager) List(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	namespace := obj.GetParent().GetID()
	k8sDaemonSets, err := getDaemonSets(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list daemonSet info failed:%s", err.Error())
		}
		return nil
	}

	var daemonSets []*types.DaemonSet
	for _, item := range k8sDaemonSets.Items {
		daemonSets = append(daemonSets, k8sDaemonSetToSCDaemonSet(&item))
	}
	return daemonSets
}

func (m *DaemonSetManager) Get(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	namespace := obj.GetParent().GetID()
	daemonSet := obj.(*types.DaemonSet)
	k8sDaemonSet, err := getDaemonSet(cluster.KubeClient, namespace, daemonSet.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get daemonSet info failed:%s", err.Error())
		}
		return nil
	}

	return k8sDaemonSetToSCDaemonSet(k8sDaemonSet)
}

func (m *DaemonSetManager) Delete(obj resttypes.Object) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := obj.GetParent().GetID()
	daemonSet := obj.(*types.DaemonSet)

	k8sDaemonSet, err := getDaemonSet(cluster.KubeClient, namespace, daemonSet.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("daemonset %s with namespace %s desn't exist", daemonSet.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
		}
	}

	if err := deleteDaemonSet(cluster.KubeClient, namespace, daemonSet.GetID()); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete daemonSet failed %s", err.Error()))
	}

	opts, ok := k8sDaemonSet.Annotations[AnnkeyForDaemonSetAdvancedoption]
	if ok {
		deleteServiceAndIngress(cluster.KubeClient, namespace, daemonSet.GetID(), opts)
	}
	return nil
}

func getDaemonSet(cli client.Client, namespace, name string) (*appsv1.DaemonSet, error) {
	daemonSet := appsv1.DaemonSet{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &daemonSet)
	return &daemonSet, err
}

func getDaemonSets(cli client.Client, namespace string) (*appsv1.DaemonSetList, error) {
	daemonSets := appsv1.DaemonSetList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &daemonSets)
	return &daemonSets, err
}

func createDaemonSet(cli client.Client, namespace string, daemonSet *types.DaemonSet) error {
	k8sPodSpec, err := scContainersToK8sPodSpec(daemonSet.Containers)
	if err != nil {
		return err
	}

	advancedOpts, _ := json.Marshal(daemonSet.AdvancedOptions)
	k8sDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonSet.Name,
			Namespace: namespace,
			Annotations: map[string]string{
				AnnkeyForDaemonSetAdvancedoption: string(advancedOpts),
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": daemonSet.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": daemonSet.Name}},
				Spec:       k8sPodSpec,
			},
		},
	}
	return cli.Create(context.TODO(), k8sDaemonSet)
}

func deleteDaemonSet(cli client.Client, namespace, name string) error {
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), daemonSet)
}

func k8sDaemonSetToSCDaemonSet(k8sDaemonSet *appsv1.DaemonSet) *types.DaemonSet {
	containers := k8sContainersToScContainers(k8sDaemonSet.Spec.Template.Spec.Containers, k8sDaemonSet.Spec.Template.Spec.Volumes)

	var conditions []types.DaemonSetCondition
	for _, condition := range k8sDaemonSet.Status.Conditions {
		conditions = append(conditions, types.DaemonSetCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: resttypes.ISOTime(condition.LastTransitionTime.Time),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	daemonSetStatus := types.DaemonSetStatus{
		CurrentNumberScheduled: k8sDaemonSet.Status.CurrentNumberScheduled,
		NumberMisscheduled:     k8sDaemonSet.Status.NumberMisscheduled,
		DesiredNumberScheduled: k8sDaemonSet.Status.DesiredNumberScheduled,
		NumberReady:            k8sDaemonSet.Status.NumberReady,
		ObservedGeneration:     k8sDaemonSet.Status.ObservedGeneration,
		UpdatedNumberScheduled: k8sDaemonSet.Status.UpdatedNumberScheduled,
		NumberAvailable:        k8sDaemonSet.Status.NumberAvailable,
		NumberUnavailable:      k8sDaemonSet.Status.NumberUnavailable,
		CollisionCount:         k8sDaemonSet.Status.CollisionCount,
		DaemonSetConditions:    conditions,
	}

	var advancedOpts types.AdvancedOptions
	opts, ok := k8sDaemonSet.Annotations[AnnkeyForDaemonSetAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	daemonSet := &types.DaemonSet{
		Name:            k8sDaemonSet.Name,
		Containers:      containers,
		AdvancedOptions: advancedOpts,
		Status:          daemonSetStatus,
	}
	daemonSet.SetID(k8sDaemonSet.Name)
	daemonSet.SetType(types.DaemonSetType)
	daemonSet.SetCreationTimestamp(k8sDaemonSet.CreationTimestamp.Time)
	return daemonSet
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type DaemonSetManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newDaemonSetManager(clusters *ClusterManager) *DaemonSetManager {
	return &DaemonSetManager{clusters: clusters}
}

func (m *DaemonSetManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	daemonSet := ctx.Object.(*types.DaemonSet)
	if err := createDaemonSet(cluster.KubeClient, namespace, daemonSet); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate daemonSet name %s", daemonSet.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create daemonSet failed %s", err.Error()))
		}
	}

	daemonSet.SetID(daemonSet.Name)
	if err := createServiceAndIngress(daemonSet.Containers, daemonSet.AdvancedOptions, cluster.KubeClient, namespace, daemonSet.Name, false); err != nil {
		deleteDaemonSet(cluster.KubeClient, namespace, daemonSet.Name)
		return nil, err
	}

	return daemonSet, nil
}

func (m *DaemonSetManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sDaemonSets, err := getDaemonSets(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list daemonSet info failed:%s", err.Error())
		}
		return nil
	}

	var daemonSets []*types.DaemonSet
	for _, item := range k8sDaemonSets.Items {
		daemonset, err := k8sDaemonSetToSCDaemonSet(cluster.KubeClient, &item)
		if err != nil {
			log.Warnf("list daemonSet info failed:%s", err.Error())
			return nil
		}
		daemonSets = append(daemonSets, daemonset)
	}
	return daemonSets
}

func (m *DaemonSetManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	daemonSet := ctx.Object.(*types.DaemonSet)
	k8sDaemonSet, err := getDaemonSet(cluster.KubeClient, namespace, daemonSet.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get daemonSet info failed:%s", err.Error())
		}
		return nil
	}

	if daemonset, err := k8sDaemonSetToSCDaemonSet(cluster.KubeClient, k8sDaemonSet); err != nil {
		log.Warnf("get daemonSet info failed:%s", err.Error())
		return nil
	} else {
		return daemonset
	}
}

func (m *DaemonSetManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	daemonSet := ctx.Object.(*types.DaemonSet)

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

	opts, ok := k8sDaemonSet.Annotations[AnnkeyForWordloadAdvancedoption]
	if ok {
		deleteServiceAndIngress(cluster.KubeClient, namespace, daemonSet.GetID(), opts)
	}
	return nil
}

func (m *DaemonSetManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	switch ctx.Action.Name {
	case types.ActionGetHistory:
		return m.getDaemonsetHistory(ctx)
	case types.ActionRollback:
		return nil, m.rollback(ctx)
	case types.ActionSetImage:
		return nil, m.setImage(ctx)
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
	}
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
	podTemplate, k8sPVCs, err := createPodTempateSpec(namespace, daemonSet, cli)
	if err != nil {
		return err
	}

	k8sDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: generatePodOwnerObjectMeta(namespace, daemonSet),
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": daemonSet.Name},
			},
			Template: *podTemplate,
		},
	}

	if err := cli.Create(context.TODO(), k8sDaemonSet); err != nil {
		deletePVCs(cli, namespace, k8sPVCs)
		return err
	}

	return nil
}

func deleteDaemonSet(cli client.Client, namespace, name string) error {
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), daemonSet)
}

func k8sDaemonSetToSCDaemonSet(cli client.Client, k8sDaemonSet *appsv1.DaemonSet) (*types.DaemonSet, error) {
	containers, templates := k8sPodSpecToScContainersAndVCTemplates(k8sDaemonSet.Spec.Template.Spec.Containers,
		k8sDaemonSet.Spec.Template.Spec.Volumes)

	pvcs, err := getPVCs(cli, k8sDaemonSet.Namespace, templates)
	if err != nil {
		return nil, err
	}

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

	var collisionCount int32
	if k8sDaemonSet.Status.CollisionCount != nil {
		collisionCount = *k8sDaemonSet.Status.CollisionCount
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
		CollisionCount:         collisionCount,
		DaemonSetConditions:    conditions,
	}

	var advancedOpts types.AdvancedOptions
	opts, ok := k8sDaemonSet.Annotations[AnnkeyForWordloadAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	daemonSet := &types.DaemonSet{
		Name:                   k8sDaemonSet.Name,
		Containers:             containers,
		AdvancedOptions:        advancedOpts,
		PersistentClaimVolumes: pvcs,
		Status:                 daemonSetStatus,
	}
	daemonSet.SetID(k8sDaemonSet.Name)
	daemonSet.SetType(types.DaemonSetType)
	daemonSet.SetCreationTimestamp(k8sDaemonSet.CreationTimestamp.Time)
	daemonSet.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sDaemonSet.Spec.Template.Annotations)
	return daemonSet, nil
}

func (m *DaemonSetManager) getDaemonsetHistory(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	daemonset := ctx.Object.(*types.DaemonSet)
	_, controllerRevisions, err := getDaemonSetAndControllerRevisions(cluster.KubeClient, namespace, daemonset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("daemonset %s with namespace %s desn't exist", daemonset.GetID(), namespace))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
		}
	}

	var versionInfos types.VersionInfos
	for _, cr := range controllerRevisions {
		var oldK8sDaemonSet appsv1.DaemonSet
		if err := json.Unmarshal(cr.Data.Raw, &oldK8sDaemonSet); err != nil {
			return nil, resttypes.NewAPIError(resttypes.InvalidFormat,
				fmt.Sprintf("unmarshal controllerrevision data failed: %v", err.Error()))
		}

		containers, _ := k8sPodSpecToScContainersAndVCTemplates(oldK8sDaemonSet.Spec.Template.Spec.Containers, nil)
		versionInfos = append(versionInfos, types.VersionInfo{
			Name:         daemonset.GetID(),
			Namespace:    namespace,
			Version:      int(cr.Revision),
			ChangeReason: cr.Annotations[ChangeCauseAnnotation],
			Containers:   containers,
		})
	}

	sort.Sort(versionInfos)
	return &types.VersionHistory{
		VersionInfos: versionInfos[:len(versionInfos)-1],
	}, nil
}

func getDaemonSetAndControllerRevisions(cli client.Client, namespace, name string) (*appsv1.DaemonSet, []appsv1.ControllerRevision, error) {
	k8sDaemonSet, err := getDaemonSet(cli, namespace, name)
	if err != nil {
		return nil, nil, err
	}

	if k8sDaemonSet.Spec.Selector == nil {
		return nil, nil, fmt.Errorf("daemonset %v has no selector", name)
	}

	controllerRevisions, err := getControllerRevisions(cli, namespace, k8sDaemonSet.Spec.Selector, k8sDaemonSet.UID)
	return k8sDaemonSet, controllerRevisions, err
}

func getControllerRevisions(cli client.Client, namespace string, selector *metav1.LabelSelector, uid k8stypes.UID) ([]appsv1.ControllerRevision, error) {
	controllerRevisionList := appsv1.ControllerRevisionList{}
	opts := &client.ListOptions{Namespace: namespace}
	labels, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	opts.LabelSelector = labels
	if err := cli.List(context.TODO(), opts, &controllerRevisionList); err != nil {
		return nil, err
	}

	var controllerRevisions []appsv1.ControllerRevision
	for _, item := range controllerRevisionList.Items {
		if isControllerBy(item.OwnerReferences, uid) {
			controllerRevisions = append(controllerRevisions, item)
		}
	}
	return controllerRevisions, nil
}

func (m *DaemonSetManager) rollback(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	daemonset := ctx.Object.(*types.DaemonSet)
	param, ok := ctx.Action.Input.(*types.RollBackVersion)
	if ok == false || param.Version < 0 {
		return resttypes.NewAPIError(resttypes.InvalidFormat,
			fmt.Sprintf("rollback version param is not valid: %v", ctx.Action.Input))
	}

	k8sDaemonSet, controllerRevisions, err := getDaemonSetAndControllerRevisions(cluster.KubeClient, namespace, daemonset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("daemonset %s with namespace %s desn't exist", daemonset.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
		}
	}

	var patch []byte
	for _, cr := range controllerRevisions {
		if int(cr.Revision) == param.Version {
			patch = cr.Data.Raw
			break
		}
	}

	if len(patch) == 0 {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("no found daemonset version: %v", param.Version))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sDaemonSet, k8stypes.StrategicMergePatchType, patch); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback daemonset failed: %v", err.Error()))
	}

	return nil
}

func (m *DaemonSetManager) setImage(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Action.Input.(*types.SetImage)
	if ok == false || len(param.Images) == 0 {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "set image param is not valid")
	}

	namespace := ctx.Object.GetParent().GetID()
	daemonset := ctx.Object.(*types.DaemonSet)
	k8sDaemonSet, err := getDaemonSet(cluster.KubeClient, namespace, daemonset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("daemonset %s with namespace %s doesn't exist", daemonset.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
		}
	}

	patch, err := getSetImagePatch(param, k8sDaemonSet.Spec.Template, k8sDaemonSet.Annotations)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("get deployment patch when set image failed: %v", err.Error()))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sDaemonSet, k8stypes.JSONPatchType, patch); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("set daemonset image failed: %v", err.Error()))
	}

	return nil
}

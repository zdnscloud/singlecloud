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
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type DaemonSetManager struct {
	clusters *ClusterManager
}

func newDaemonSetManager(clusters *ClusterManager) *DaemonSetManager {
	return &DaemonSetManager{clusters: clusters}
}

func (m *DaemonSetManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	daemonSet := ctx.Resource.(*types.DaemonSet)
	if err := createDaemonSet(cluster.KubeClient, namespace, daemonSet); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate daemonSet name %s", daemonSet.Name))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create daemonSet failed %s", err.Error()))
		}
	}

	daemonSet.SetID(daemonSet.Name)
	return daemonSet, nil
}

func (m *DaemonSetManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
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

func (m *DaemonSetManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	daemonSet := ctx.Resource.(*types.DaemonSet)
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

func (m *DaemonSetManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	daemonSet := ctx.Resource.(*types.DaemonSet)
	k8sDaemonSet, err := getDaemonSet(cluster.KubeClient, namespace, daemonSet.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("daemonset %s desn't exist", namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
		}
	}

	k8sPodSpec, _, err := scPodSpecToK8sPodSpecAndPVCs(daemonSet.Containers, daemonSet.PersistentVolumes)
	if err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update daemonset failed %s", err.Error()))
	}

	k8sDaemonSet.Spec.Template.Spec = k8sPodSpec
	k8sDaemonSet.Annotations[ChangeCauseAnnotation] = daemonSet.Memo
	if err := cluster.KubeClient.Update(context.TODO(), k8sDaemonSet); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update daemonset failed %s", err.Error()))
	}

	return daemonSet, nil
}

func (m *DaemonSetManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	daemonSet := ctx.Resource.(*types.DaemonSet)

	k8sDaemonSet, err := getDaemonSet(cluster.KubeClient, namespace, daemonSet.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("daemonset %s with namespace %s desn't exist", daemonSet.GetID(), namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
		}
	}

	if err := deleteDaemonSet(cluster.KubeClient, namespace, daemonSet.GetID()); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete daemonSet failed %s", err.Error()))
	}

	if delete, ok := k8sDaemonSet.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cluster.KubeClient, namespace, k8sDaemonSet.Spec.Template.Spec.Volumes)
	}
	return nil
}

func (m *DaemonSetManager) Action(ctx *resource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case types.ActionGetHistory:
		return m.getDaemonsetHistory(ctx)
	case types.ActionRollback:
		return nil, m.rollback(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
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

	pvs, err := getPVCs(cli, k8sDaemonSet.Namespace, templates)
	if err != nil {
		return nil, err
	}

	var collisionCount int
	if k8sDaemonSet.Status.CollisionCount != nil {
		collisionCount = int(*k8sDaemonSet.Status.CollisionCount)
	}

	daemonSetStatus := types.DaemonSetStatus{
		CurrentNumberScheduled: int(k8sDaemonSet.Status.CurrentNumberScheduled),
		NumberMisscheduled:     int(k8sDaemonSet.Status.NumberMisscheduled),
		DesiredNumberScheduled: int(k8sDaemonSet.Status.DesiredNumberScheduled),
		NumberReady:            int(k8sDaemonSet.Status.NumberReady),
		ObservedGeneration:     int(k8sDaemonSet.Status.ObservedGeneration),
		UpdatedNumberScheduled: int(k8sDaemonSet.Status.UpdatedNumberScheduled),
		NumberAvailable:        int(k8sDaemonSet.Status.NumberAvailable),
		NumberUnavailable:      int(k8sDaemonSet.Status.NumberUnavailable),
		CollisionCount:         collisionCount,
		Conditions:             k8sWorkloadConditionsToScWorkloadConditions(k8sDaemonSet.Status.Conditions, false),
	}

	var advancedOpts types.AdvancedOptions
	opts, ok := k8sDaemonSet.Annotations[AnnkeyForWordloadAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	daemonSet := &types.DaemonSet{
		Name:              k8sDaemonSet.Name,
		Containers:        containers,
		AdvancedOptions:   advancedOpts,
		PersistentVolumes: pvs,
		Status:            daemonSetStatus,
	}
	daemonSet.SetID(k8sDaemonSet.Name)
	daemonSet.SetCreationTimestamp(k8sDaemonSet.CreationTimestamp.Time)
	if k8sDaemonSet.GetDeletionTimestamp() != nil {
		daemonSet.SetDeletionTimestamp(k8sDaemonSet.DeletionTimestamp.Time)
	}
	daemonSet.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sDaemonSet.Spec.Template.Annotations)
	return daemonSet, nil
}

func (m *DaemonSetManager) getDaemonsetHistory(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	daemonset := ctx.Resource.(*types.DaemonSet)
	_, controllerRevisions, err := getDaemonSetAndControllerRevisions(cluster.KubeClient, namespace, daemonset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("daemonset %s with namespace %s desn't exist", daemonset.GetID(), namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
		}
	}

	var versionInfos types.VersionInfos
	for _, cr := range controllerRevisions {
		var oldK8sDaemonSet appsv1.DaemonSet
		if err := json.Unmarshal(cr.Data.Raw, &oldK8sDaemonSet); err != nil {
			return nil, resterror.NewAPIError(resterror.InvalidFormat,
				fmt.Sprintf("unmarshal controllerrevision data failed: %v", err.Error()))
		}

		containers, _ := k8sPodSpecToScContainersAndVCTemplates(oldK8sDaemonSet.Spec.Template.Spec.Containers,
			oldK8sDaemonSet.Spec.Template.Spec.Volumes)
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

func (m *DaemonSetManager) rollback(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	daemonset := ctx.Resource.(*types.DaemonSet)
	param, ok := ctx.Resource.GetAction().Input.(*types.RollBackVersion)
	if ok == false {
		return resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("action rollback version param is not valid"))
	}

	k8sDaemonSet, controllerRevisions, err := getDaemonSetAndControllerRevisions(cluster.KubeClient, namespace, daemonset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("daemonset %s with namespace %s desn't exist", daemonset.GetID(), namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get daemonset failed %s", err.Error()))
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
		return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found daemonset version: %v", param.Version))
	}

	//TODO add update ControllerRevision.Annotations[kubernetes.io/change-cause] with new memo, now memo is readonly
	if err := cluster.KubeClient.Patch(context.TODO(), k8sDaemonSet, k8stypes.StrategicMergePatchType, patch); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback daemonset failed: %v", err.Error()))
	}

	return nil
}

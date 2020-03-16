package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type StatefulSetManager struct {
	clusters *ClusterManager
}

func newStatefulSetManager(clusters *ClusterManager) *StatefulSetManager {
	return &StatefulSetManager{clusters: clusters}
}

func (m *StatefulSetManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	statefulset := ctx.Resource.(*types.StatefulSet)
	if err := createStatefulSet(cluster.GetKubeClient(), namespace, statefulset); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate statefulset name %s", statefulset.Name))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create statefulset failed %s", err.Error()))
	}

	statefulset.SetID(statefulset.Name)
	return statefulset, nil
}

func (m *StatefulSetManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sStatefulSets, err := getStatefulSets(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found statefulsets")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list statefulsets failed %s", err.Error()))
	}

	var statefulsets []*types.StatefulSet
	for _, statefulset := range k8sStatefulSets.Items {
		statefulsets = append(statefulsets, k8sStatefulSetToSCStatefulSet(&statefulset))
	}
	return statefulsets, nil
}

func (m *StatefulSetManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	statefulset := ctx.Resource.(*types.StatefulSet)
	k8sStatefulSet, err := getStatefulSet(cluster.GetKubeClient(), namespace, statefulset.GetID())
	if err != nil {
		return nil, err
	}

	return k8sStatefulSetToSCStatefulSet(k8sStatefulSet), nil
}

func (m *StatefulSetManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	statefulSet := ctx.Resource.(*types.StatefulSet)
	k8sStatefulSet, apiErr := getStatefulSet(cluster.GetKubeClient(), namespace, statefulSet.GetID())
	if apiErr != nil {
		return nil, apiErr
	}

	k8sPodSpec, _, err := scPodSpecToK8sPodSpecAndPVCs(statefulSet.Containers, statefulSet.PersistentVolumes)
	if err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update statefulset failed %s", err.Error()))
	}

	k8sStatefulSet.Spec.Template.Spec.Containers = k8sPodSpec.Containers
	k8sStatefulSet.Spec.Template.Spec.Volumes = k8sPodSpec.Volumes
	k8sStatefulSet.Annotations = addWorkloadUpdateMemoToAnnotations(k8sStatefulSet.Annotations, statefulSet.Memo)
	if err := cluster.GetKubeClient().Update(context.TODO(), k8sStatefulSet); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update statefulset failed %s", err.Error()))
	}

	return statefulSet, nil
}

func (m *StatefulSetManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	statefulset := ctx.Resource.(*types.StatefulSet)

	k8sStatefulSet, apiErr := getStatefulSet(cluster.GetKubeClient(), namespace, statefulset.GetID())
	if apiErr != nil {
		return apiErr
	}

	volumes, err := getStatefulSetPodsVolumes(cluster.GetKubeClient(), namespace, k8sStatefulSet)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found statefulset %s pods", statefulset.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get statefulset %s pods volumes failed %s", statefulset.GetID(), err.Error()))
	}

	if err := deleteStatefulSet(cluster.GetKubeClient(), namespace, statefulset.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete statefulset %s failed %s", statefulset.GetID(), err.Error()))
	}

	if delete, ok := k8sStatefulSet.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cluster.GetKubeClient(), namespace, volumes)
	}
	eb.PublishResourceDeleteEvent(statefulset)
	return nil
}

func (m *StatefulSetManager) Action(ctx *resource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case types.ActionGetHistory:
		return m.getStatefulSetHistory(ctx)
	case types.ActionRollback:
		return nil, m.rollback(ctx)
	case types.ActionSetPodCount:
		return m.setPodCount(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func getStatefulSetPodsVolumes(cli client.Client, namespace string, k8sStatefulSet *appsv1.StatefulSet) ([]corev1.Volume, error) {
	k8sPods, err := getOwnerPods(cli, namespace, types.ResourceTypeStatefulSet, k8sStatefulSet.Name)
	if err != nil {
		return nil, err
	}

	var volumes []corev1.Volume
	for _, item := range k8sPods.Items {
		volumes = append(volumes, item.Spec.Volumes...)
	}

	return volumes, nil
}

func getStatefulSet(cli client.Client, namespace, name string) (*appsv1.StatefulSet, *resterror.APIError) {
	statefulset := appsv1.StatefulSet{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &statefulset); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found statefulSet %s", name))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get statefulSet %s failed %s", name, err.Error()))
	}

	return &statefulset, nil
}

func getStatefulSets(cli client.Client, namespace string) (*appsv1.StatefulSetList, error) {
	statefulsets := appsv1.StatefulSetList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &statefulsets)
	return &statefulsets, err
}

func createStatefulSet(cli client.Client, namespace string, statefulset *types.StatefulSet) error {
	podTemplate, k8sPVCs, err := createPodTempateSpec(namespace, statefulset, cli)
	if err != nil {
		return err
	}

	replicas := int32(statefulset.Replicas)
	k8sStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: generatePodOwnerObjectMeta(namespace, statefulset),
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: statefulset.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": statefulset.Name},
			},
			Template:             *podTemplate,
			VolumeClaimTemplates: k8sPVCs,
		},
	}
	return cli.Create(context.TODO(), k8sStatefulSet)
}

func deleteStatefulSet(cli client.Client, namespace, name string) error {
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), statefulset)
}

func k8sStatefulSetToSCStatefulSet(k8sStatefulSet *appsv1.StatefulSet) *types.StatefulSet {
	var advancedOpts types.AdvancedOptions
	if opts, ok := k8sStatefulSet.Annotations[AnnkeyForWordloadAdvancedoption]; ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	containers, templates := k8sPodSpecToScContainersAndVCTemplates(k8sStatefulSet.Spec.Template.Spec.Containers,
		k8sStatefulSet.Spec.Template.Spec.Volumes)

	var pvs []types.PersistentVolumeTemplate
	for _, template := range templates {
		if template.StorageClassName == types.StorageClassNameTemp {
			pvs = append(pvs, template)
		}
	}

	for _, pvc := range k8sStatefulSet.Spec.VolumeClaimTemplates {
		if pvc.Spec.StorageClassName != nil {
			quantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
			pvs = append(pvs, types.PersistentVolumeTemplate{
				Name:             pvc.Name,
				Size:             quantity.String(),
				StorageClassName: *pvc.Spec.StorageClassName,
			})
		}
	}

	statefulset := &types.StatefulSet{
		Name:              k8sStatefulSet.Name,
		Replicas:          int(*k8sStatefulSet.Spec.Replicas),
		Containers:        containers,
		AdvancedOptions:   advancedOpts,
		PersistentVolumes: pvs,
		Status:            types.WorkloadStatus{ReadyReplicas: int(k8sStatefulSet.Status.ReadyReplicas)},
	}

	if k8sStatefulSet.Status.CurrentRevision != k8sStatefulSet.Status.UpdateRevision {
		statefulset.Status.Updating = true
		statefulset.Status.UpdatingReplicas = 1
		if k8sStatefulSet.Status.UpdatedReplicas != 0 {
			statefulset.Status.UpdatedReplicas = int(k8sStatefulSet.Status.UpdatedReplicas - 1)
			statefulset.Status.CurrentReplicas = int(*k8sStatefulSet.Spec.Replicas - k8sStatefulSet.Status.UpdatedReplicas)
		} else {
			statefulset.Status.CurrentReplicas = int(*k8sStatefulSet.Spec.Replicas - 1)
		}
	}

	statefulset.Status.Conditions = k8sWorkloadConditionsToScWorkloadConditions(k8sStatefulSet.Status.Conditions, false)
	statefulset.SetID(k8sStatefulSet.Name)
	statefulset.SetCreationTimestamp(k8sStatefulSet.CreationTimestamp.Time)
	if k8sStatefulSet.GetDeletionTimestamp() != nil {
		statefulset.SetDeletionTimestamp(k8sStatefulSet.DeletionTimestamp.Time)
	}
	statefulset.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sStatefulSet.Spec.Template.Annotations)
	return statefulset
}

func (m *StatefulSetManager) getStatefulSetHistory(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	statefulset := ctx.Resource.(*types.StatefulSet)
	_, controllerRevisions, err := getStatefulSetAndControllerRevisions(cluster.GetKubeClient(), namespace, statefulset.GetID())
	if err != nil {
		return nil, err
	}

	var versionInfos types.VersionInfos
	for _, cr := range controllerRevisions {
		oldK8sStatefulSet := appsv1.StatefulSet{}
		if err := json.Unmarshal(cr.Data.Raw, &oldK8sStatefulSet); err != nil {
			return nil, resterror.NewAPIError(resterror.InvalidFormat,
				fmt.Sprintf("unmarshal controllerrevision data failed: %v", err.Error()))
		}
		containers, _ := k8sPodSpecToScContainersAndVCTemplates(oldK8sStatefulSet.Spec.Template.Spec.Containers,
			oldK8sStatefulSet.Spec.Template.Spec.Volumes)
		versionInfos = append(versionInfos, types.VersionInfo{
			Name:         statefulset.GetID(),
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

func getStatefulSetAndControllerRevisions(cli client.Client, namespace, name string) (*appsv1.StatefulSet, []appsv1.ControllerRevision, *resterror.APIError) {
	k8sStatefulSet, apiErr := getStatefulSet(cli, namespace, name)
	if apiErr != nil {
		return nil, nil, apiErr
	}

	if k8sStatefulSet.Spec.Selector == nil {
		return nil, nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("statefulset %s has no selector", name))
	}

	controllerRevisions, err := getControllerRevisions(cli, namespace, k8sStatefulSet.Spec.Selector, k8sStatefulSet.UID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found statefulset %s controllerRevisions", name))
		}
		return nil, nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get statefulset %s controllerRevisions failed: %s", name, err.Error()))
	}

	return k8sStatefulSet, controllerRevisions, nil
}

func (m *StatefulSetManager) rollback(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	statefulset := ctx.Resource.(*types.StatefulSet)
	param, ok := ctx.Resource.GetAction().Input.(*types.RollBackVersion)
	if ok == false {
		return resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("action rollback version param is not valid"))
	}

	k8sStatefulSet, controllerRevisions, err := getStatefulSetAndControllerRevisions(cluster.GetKubeClient(), namespace, statefulset.GetID())
	if err != nil {
		return err
	}

	var patch []byte
	for _, cr := range controllerRevisions {
		if int(cr.Revision) == param.Version {
			patch = cr.Data.Raw
			break
		}
	}

	if len(patch) == 0 {
		return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found statefulset version: %v", param.Version))
	}

	if err := cluster.GetKubeClient().Patch(context.TODO(), k8sStatefulSet, k8stypes.StrategicMergePatchType, patch); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback statefulset failed: %v", err.Error()))
	}

	return nil
}

func (m *StatefulSetManager) setPodCount(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Resource.GetAction().Input.(*types.SetPodCount)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, "action set pod count param is not valid")
	}

	namespace := ctx.Resource.GetParent().GetID()
	statefulset := ctx.Resource.(*types.StatefulSet)
	k8sStatefulSet, err := getStatefulSet(cluster.GetKubeClient(), namespace, statefulset.GetID())
	if err != nil {
		return nil, err
	}

	if int(*k8sStatefulSet.Spec.Replicas) != param.Replicas {
		if err := cluster.GetKubeClient().Patch(context.TODO(), k8sStatefulSet, k8stypes.MergePatchType,
			[]byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, param.Replicas))); err != nil {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("set statefulset pod count failed: %v", err.Error()))
		}
	}

	return param, nil
}

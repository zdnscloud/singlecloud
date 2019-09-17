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

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest"
	resttypes "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type StatefulSetManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newStatefulSetManager(clusters *ClusterManager) *StatefulSetManager {
	return &StatefulSetManager{clusters: clusters}
}

func (m *StatefulSetManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	statefulset := ctx.Object.(*types.StatefulSet)
	if err := createStatefulSet(cluster.KubeClient, namespace, statefulset); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate statefulset name %s", statefulset.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create statefulset failed %s", err.Error()))
		}
	}

	statefulset.SetID(statefulset.Name)
	return statefulset, nil
}

func (m *StatefulSetManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sStatefulSets, err := getStatefulSets(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list statefulset info failed:%s", err.Error())
		}
		return nil
	}

	var statefulsets []*types.StatefulSet
	for _, statefulset := range k8sStatefulSets.Items {
		statefulsets = append(statefulsets, k8sStatefulSetToSCStatefulSet(&statefulset))
	}
	return statefulsets
}

func (m *StatefulSetManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	statefulset := ctx.Object.(*types.StatefulSet)
	k8sStatefulSet, err := getStatefulSet(cluster.KubeClient, namespace, statefulset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get statefulset info failed:%s", err.Error())
		}
		return nil
	}

	return k8sStatefulSetToSCStatefulSet(k8sStatefulSet)
}

func (m *StatefulSetManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	statefulset := ctx.Object.(*types.StatefulSet)

	k8sStatefulSet, err := getStatefulSet(cluster.KubeClient, namespace, statefulset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("statefulset in namespace %s is non-exist", namespace))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get statefulset failed %s", err.Error()))
		}
	}

	if int(*k8sStatefulSet.Spec.Replicas) == statefulset.Replicas {
		return statefulset, nil
	} else {
		replicas := int32(statefulset.Replicas)
		k8sStatefulSet.Spec.Replicas = &replicas
		err := cluster.KubeClient.Update(context.TODO(), k8sStatefulSet)
		if err != nil {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update statefulset failed %s", err.Error()))
		} else {
			return statefulset, nil
		}
	}
}

func (m *StatefulSetManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	statefulset := ctx.Object.(*types.StatefulSet)

	k8sStatefulSet, err := getStatefulSet(cluster.KubeClient, namespace, statefulset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("statefulset in namespace %s is non-exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get statefulset failed %s", err.Error()))
		}
	}

	volumes, err := getStatefulSetPodsVolumes(cluster.KubeClient, namespace, k8sStatefulSet)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get statefulset volumes failed %s", err.Error()))
	}

	if err := deleteStatefulSet(cluster.KubeClient, namespace, statefulset.GetID()); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete statefulset failed %s", err.Error()))
	}

	if delete, ok := k8sStatefulSet.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cluster.KubeClient, namespace, volumes)
	}
	return nil
}

func (m *StatefulSetManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	switch ctx.Action.Name {
	case types.ActionGetHistory:
		return m.getStatefulSetHistory(ctx)
	case types.ActionRollback:
		return nil, m.rollback(ctx)
	case types.ActionSetImage:
		return nil, m.setImage(ctx)
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
	}
}

func getStatefulSetPodsVolumes(cli client.Client, namespace string, k8sStatefulSet *appsv1.StatefulSet) ([]corev1.Volume, error) {
	k8sPods, err := getOwnerPods(cli, namespace, types.StatefulSetType, k8sStatefulSet.Name)
	if err != nil {
		return nil, err
	}

	var volumes []corev1.Volume
	for _, item := range k8sPods.Items {
		volumes = append(volumes, item.Spec.Volumes...)
	}

	return volumes, nil
}

func getStatefulSet(cli client.Client, namespace, name string) (*appsv1.StatefulSet, error) {
	statefulset := appsv1.StatefulSet{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &statefulset)
	return &statefulset, err
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
	opts, ok := k8sStatefulSet.Annotations[AnnkeyForWordloadAdvancedoption]
	if ok {
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
	}
	statefulset.SetID(k8sStatefulSet.Name)
	statefulset.SetType(types.StatefulSetType)
	statefulset.SetCreationTimestamp(k8sStatefulSet.CreationTimestamp.Time)
	statefulset.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sStatefulSet.Spec.Template.Annotations)
	return statefulset
}

func (m *StatefulSetManager) getStatefulSetHistory(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	statefulset := ctx.Object.(*types.StatefulSet)
	_, controllerRevisions, err := getStatefulSetAndControllerRevisions(cluster.KubeClient, namespace, statefulset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("statefulset %s with namespace %s doesn't exist", statefulset.GetID(), namespace))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get statefulset failed %s", err.Error()))
		}
	}

	var versionInfos types.VersionInfos
	for _, cr := range controllerRevisions {
		oldK8sStatefulSet := appsv1.StatefulSet{}
		if err := json.Unmarshal(cr.Data.Raw, &oldK8sStatefulSet); err != nil {
			return nil, resttypes.NewAPIError(resttypes.InvalidFormat,
				fmt.Sprintf("unmarshal controllerrevision data failed: %v", err.Error()))
		}
		containers, _ := k8sPodSpecToScContainersAndVCTemplates(oldK8sStatefulSet.Spec.Template.Spec.Containers, nil)
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

func getStatefulSetAndControllerRevisions(cli client.Client, namespace, name string) (*appsv1.StatefulSet, []appsv1.ControllerRevision, error) {
	k8sStatefulSet, err := getStatefulSet(cli, namespace, name)
	if err != nil {
		return nil, nil, err
	}

	if k8sStatefulSet.Spec.Selector == nil {
		return nil, nil, fmt.Errorf("statefulset %v has no selector", name)
	}

	controllerRevisions, err := getControllerRevisions(cli, namespace, k8sStatefulSet.Spec.Selector, k8sStatefulSet.UID)
	return k8sStatefulSet, controllerRevisions, err
}

func (m *StatefulSetManager) rollback(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	statefulset := ctx.Object.(*types.StatefulSet)
	param, ok := ctx.Action.Input.(*types.RollBackVersion)
	if ok == false || param.Version < 0 {
		return resttypes.NewAPIError(resttypes.InvalidFormat,
			fmt.Sprintf("rollback version param is not valid: %v", ctx.Action.Input))
	}

	k8sStatefulSet, controllerRevisions, err := getStatefulSetAndControllerRevisions(cluster.KubeClient, namespace, statefulset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("statefulset %s with namespace %s desn't exist", statefulset.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get statefulset failed %s", err.Error()))
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
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("no found statefulset version: %v", param.Version))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sStatefulSet, k8stypes.StrategicMergePatchType, patch); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback statefulset failed: %v", err.Error()))
	}

	return nil
}

func (m *StatefulSetManager) setImage(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Action.Input.(*types.SetImage)
	if ok == false || len(param.Images) == 0 {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "set image param is not valid")
	}

	namespace := ctx.Object.GetParent().GetID()
	statefulset := ctx.Object.(*types.StatefulSet)
	k8sStatefulSet, err := getStatefulSet(cluster.KubeClient, namespace, statefulset.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("statefulset %s with namespace %s doesn't exist", statefulset.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get statefulset failed %s", err.Error()))
		}
	}

	patch, err := getSetImagePatch(param, k8sStatefulSet.Spec.Template, k8sStatefulSet.Annotations)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("get statefulset patch when set image failed: %v", err.Error()))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sStatefulSet, k8stypes.JSONPatchType, patch); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("set statefulset image failed: %v", err.Error()))
	}

	return nil
}

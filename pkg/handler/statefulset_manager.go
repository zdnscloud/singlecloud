package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AnnkeyForStatefulSetAdvancedoption = "zcloud_statefulsetment_advanded_options"
	StatefulSetPVCNameConnector        = "-pvc"
)

var FilesystemVolumeMode = corev1.PersistentVolumeFilesystem

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
	containerPorts := make(map[string]types.ContainerPort)
	for _, container := range statefulset.Containers {
		for _, port := range container.ExposedPorts {
			containerPorts[port.Name] = port
		}
	}
	if err := createServiceAndIngress(containerPorts, statefulset.AdvancedOptions, cluster.KubeClient, namespace, statefulset.Name, true); err != nil {
		deleteStatefulSet(cluster.KubeClient, namespace, statefulset.Name)
		return nil, err
	}

	advancedOpts, _ := json.Marshal(statefulset.AdvancedOptions)
	statefulset.SetID(statefulset.Name)
	if err := createStatefulSet(cluster.KubeClient, namespace, statefulset, string(advancedOpts)); err != nil {
		deleteServiceAndIngress(cluster.KubeClient, namespace, statefulset.GetID(), string(advancedOpts))
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate statefulset name %s", statefulset.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create statefulset failed %s", err.Error()))
		}
	}

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

	if err := deleteStatefulSet(cluster.KubeClient, namespace, statefulset.GetID()); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete statefulset failed %s", err.Error()))
	}

	opts, ok := k8sStatefulSet.Annotations[AnnkeyForStatefulSetAdvancedoption]
	if ok {
		deleteServiceAndIngress(cluster.KubeClient, namespace, statefulset.GetID(), opts)
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

func createStatefulSet(cli client.Client, namespace string, statefulset *types.StatefulSet, opts string) error {
	replicas := int32(statefulset.Replicas)
	k8sPodSpec, err := scContainersToK8sPodSpec(statefulset.Containers)
	if err != nil {
		return err
	}

	k8sVolumes, k8sVolumeClaimTemplates, err := scPVCToK8sVolumesAndPVCs(statefulset.VolumeClaimTemplate)
	if err != nil {
		return err
	}

	k8sPodSpec.Volumes = append(k8sPodSpec.Volumes, k8sVolumes...)
	for i, _ := range k8sPodSpec.Containers {
		k8sPodSpec.Containers[i].VolumeMounts = append(k8sPodSpec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      statefulset.VolumeClaimTemplate.Name,
			MountPath: statefulset.VolumeClaimTemplate.MountPath,
		})
	}

	k8sStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulset.Name,
			Namespace: namespace,
			Annotations: map[string]string{
				AnnkeyForStatefulSetAdvancedoption: opts,
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: statefulset.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": statefulset.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: scExposedMetricToK8sTempateObjectMeta(statefulset.Name, statefulset.AdvancedOptions.ExposedMetric),
				Spec:       k8sPodSpec,
			},
			VolumeClaimTemplates: k8sVolumeClaimTemplates,
		},
	}
	return cli.Create(context.TODO(), k8sStatefulSet)
}

func scPVCToK8sVolumesAndPVCs(pvc types.VolumeClaimTemplate) ([]corev1.Volume, []corev1.PersistentVolumeClaim, error) {
	if pvc.StorageClassName == "" {
		return nil, nil, nil
	}

	var k8sQuantity *resource.Quantity
	if pvc.StorageSize != "" {
		quantity, err := resource.ParseQuantity(pvc.StorageSize)
		if err != nil {
			return nil, nil, fmt.Errorf("parse statefulset storageSize %s failed: %s", pvc.StorageSize, err.Error())
		}
		k8sQuantity = &quantity
	}

	var accessModes []corev1.PersistentVolumeAccessMode
	switch pvc.StorageClassName {
	case types.StorageClassNameTemp:
		var k8sVolumes []corev1.Volume
		k8sVolumes = append(k8sVolumes, corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: k8sQuantity,
				},
			},
		})
		return k8sVolumes, nil, nil
	case types.StorageClassNameLVM:
		accessModes = append(accessModes, corev1.ReadWriteOnce)
	case types.StorageClassNameNFS:
		accessModes = append(accessModes, corev1.ReadWriteMany)
	default:
		return nil, nil, fmt.Errorf("statefulset volumeclaimtemplate storageclass %s isn`t supported", pvc.StorageClassName)
	}

	if k8sQuantity == nil {
		return nil, nil, fmt.Errorf("statefulset volumeClaimTemplates storageSize must not be zero")
	}

	var pvcs []corev1.PersistentVolumeClaim
	pvcs = append(pvcs, corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvc.Name,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *k8sQuantity,
				},
			},
			StorageClassName: &pvc.StorageClassName,
			VolumeMode:       &FilesystemVolumeMode,
		},
	})

	return nil, pvcs, nil
}

func deleteStatefulSet(cli client.Client, namespace, name string) error {
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), statefulset)
}

func k8sStatefulSetToSCStatefulSet(k8sStatefulSet *appsv1.StatefulSet) *types.StatefulSet {
	containers := k8sContainersToScContainers(k8sStatefulSet.Spec.Template.Spec.Containers, k8sStatefulSet.Spec.Template.Spec.Volumes)

	var advancedOpts types.AdvancedOptions
	opts, ok := k8sStatefulSet.Annotations[AnnkeyForStatefulSetAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	var volumeClaimTemplate types.VolumeClaimTemplate
	isEmptyDir := false
	for _, volume := range k8sStatefulSet.Spec.Template.Spec.Volumes {
		if volume.VolumeSource.EmptyDir != nil {
			isEmptyDir = true
			volumeClaimTemplate = types.VolumeClaimTemplate{
				Name:             volume.Name,
				StorageClassName: types.StorageClassNameTemp,
			}
			if volume.VolumeSource.EmptyDir.SizeLimit != nil {
				volumeClaimTemplate.StorageSize = volume.VolumeSource.EmptyDir.SizeLimit.String()
			}
			break
		}
	}

	if isEmptyDir == false {
		for _, pvc := range k8sStatefulSet.Spec.VolumeClaimTemplates {
			if pvc.Spec.StorageClassName != nil {
				quantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
				volumeClaimTemplate = types.VolumeClaimTemplate{
					Name:             pvc.Name,
					StorageSize:      quantity.String(),
					StorageClassName: *pvc.Spec.StorageClassName,
				}
				break
			}
		}
	}

	for _, container := range k8sStatefulSet.Spec.Template.Spec.Containers {
		for _, mount := range container.VolumeMounts {
			if mount.Name == volumeClaimTemplate.Name {
				volumeClaimTemplate.MountPath = mount.MountPath
				break
			}
		}
	}

	statefulset := &types.StatefulSet{
		Name:                k8sStatefulSet.Name,
		Replicas:            int(*k8sStatefulSet.Spec.Replicas),
		Containers:          containers,
		AdvancedOptions:     advancedOpts,
		VolumeClaimTemplate: volumeClaimTemplate,
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
		versionInfos = append(versionInfos, types.VersionInfo{
			Name:         statefulset.GetID(),
			Namespace:    namespace,
			Version:      int(cr.Revision),
			ChangeReason: cr.Annotations[ChangeCauseAnnotation],
			Containers:   k8sContainersToScContainers(oldK8sStatefulSet.Spec.Template.Spec.Containers, nil),
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

package handler

import (
	"context"
	"encoding/json"
	"fmt"

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
	k8sVolume, k8sVolumeClaimTemplates, err := scPVCToK8sVolumeAndPVCs(statefulset.VolumeClaimTemplate)
	if err != nil {
		return err
	}

	k8sPodSpec, err := scContainersToK8sPodSpec(statefulset.Containers, k8sVolume)
	if err != nil {
		return err
	}

	replicas := int32(statefulset.Replicas)
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

func scPVCToK8sVolumeAndPVCs(pvc types.VolumeClaimTemplate) (*corev1.Volume, []corev1.PersistentVolumeClaim, error) {
	if pvc.StorageClassName == "" {
		return nil, nil, nil
	}

	var k8sQuantity *resource.Quantity
	if pvc.Size != "" {
		quantity, err := resource.ParseQuantity(pvc.Size)
		if err != nil {
			return nil, nil, fmt.Errorf("parse statefulset storageSize %s failed: %s", pvc.Size, err.Error())
		}
		k8sQuantity = &quantity
	}

	var accessModes []corev1.PersistentVolumeAccessMode
	switch pvc.StorageClassName {
	case types.StorageClassNameTemp:
		return &corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: k8sQuantity,
				},
			},
		}, nil, nil
	case types.StorageClassNameLVM:
		accessModes = append(accessModes, corev1.ReadWriteOnce)
	case types.StorageClassNameNFS:
		fallthrough
	case types.StorageClassNameCeph:
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
	var advancedOpts types.AdvancedOptions
	opts, ok := k8sStatefulSet.Annotations[AnnkeyForStatefulSetAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	containers, volumeClaimTemplate := k8sContainersToScContainersAndPVCTemplate(k8sStatefulSet.Spec.Template.Spec.Containers,
		k8sStatefulSet.Spec.Template.Spec.Volumes)

	if volumeClaimTemplate.StorageClassName == "" {
		for _, pvc := range k8sStatefulSet.Spec.VolumeClaimTemplates {
			if pvc.Spec.StorageClassName != nil {
				quantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
				volumeClaimTemplate = types.VolumeClaimTemplate{
					Name:             pvc.Name,
					Size:             quantity.String(),
					StorageClassName: *pvc.Spec.StorageClassName,
				}
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

package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type PersistentVolumeManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newPersistentVolumeManager(clusters *ClusterManager) *PersistentVolumeManager {
	return &PersistentVolumeManager{clusters: clusters}
}

func (m *PersistentVolumeManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	k8sPersistentVolumes, err := getPersistentVolumes(cluster.KubeClient)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list persistentvolume info failed:%s", err.Error())
		}
		return nil
	}

	var pvs []*types.PersistentVolume
	for _, item := range k8sPersistentVolumes.Items {
		pvs = append(pvs, k8sPVToSCPV(&item))
	}
	return pvs
}

func (m PersistentVolumeManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	pv := ctx.Object.(*types.PersistentVolume)
	k8sPersistentVolume, err := getPersistentVolume(cluster.KubeClient, pv.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get persistentvolume info failed:%s", err.Error())
		}
		return nil
	}

	return k8sPVToSCPV(k8sPersistentVolume)
}

func (m PersistentVolumeManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	pv := ctx.Object.(*types.PersistentVolume)
	err := deletePersistentVolume(cluster.KubeClient, pv.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("persistentvolume %s doesn't exist", pv.GetID()))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete persistentvolume failed %s", err.Error()))
		}
	}
	return nil
}

func getPersistentVolume(cli client.Client, name string) (*corev1.PersistentVolume, error) {
	pv := corev1.PersistentVolume{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &pv)
	return &pv, err
}

func getPersistentVolumes(cli client.Client) (*corev1.PersistentVolumeList, error) {
	pvs := corev1.PersistentVolumeList{}
	err := cli.List(context.TODO(), nil, &pvs)
	return &pvs, err
}

func deletePersistentVolume(cli client.Client, name string) error {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), pv)
}

func k8sPVToSCPV(k8sPersistentVolume *corev1.PersistentVolume) *types.PersistentVolume {
	var ref types.ClaimRef
	if claim := k8sPersistentVolume.Spec.ClaimRef; claim != nil {
		ref = types.ClaimRef{
			Kind:      claim.Kind,
			Name:      claim.Name,
			Namespace: claim.Namespace,
		}
	}

	var capacity string
	if quantity, ok := k8sPersistentVolume.Spec.Capacity[corev1.ResourceStorage]; ok {
		capacity = quantity.String()
	}

	pv := &types.PersistentVolume{
		Name:             k8sPersistentVolume.Name,
		StorageSize:      capacity,
		StorageClassName: k8sPersistentVolume.Spec.StorageClassName,
		ClaimRef:         ref,
		Status:           string(k8sPersistentVolume.Status.Phase),
	}
	pv.SetID(k8sPersistentVolume.Name)
	pv.SetType(types.PersistentVolumeType)
	pv.SetCreationTimestamp(k8sPersistentVolume.CreationTimestamp.Time)
	return pv
}

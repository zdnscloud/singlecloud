package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

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

const (
	ChangeCauseAnnotation       = "kubernetes.io/change-cause"
	LastAppliedConfigAnnotation = "kubectl.kubernetes.io/last-applied-configuration"
	RevisionAnnotation          = "deployment.kubernetes.io/revision"
	RevisionHistoryAnnotation   = "deployment.kubernetes.io/revision-history"
	DesiredReplicasAnnotation   = "deployment.kubernetes.io/desired-replicas"
	MaxReplicasAnnotation       = "deployment.kubernetes.io/max-replicas"
	DeprecatedRollbackTo        = "deprecated.deployment.rollback.to"
)

var AnnotationsToSkip = map[string]bool{
	LastAppliedConfigAnnotation: true,
	RevisionAnnotation:          true,
	RevisionHistoryAnnotation:   true,
	DesiredReplicasAnnotation:   true,
	MaxReplicasAnnotation:       true,
	DeprecatedRollbackTo:        true,
}

type DeploymentManager struct {
	clusters *ClusterManager
}

func newDeploymentManager(clusters *ClusterManager) *DeploymentManager {
	return &DeploymentManager{clusters: clusters}
}

func (m *DeploymentManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	if err := createDeployment(cluster.KubeClient, namespace, deploy); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate deploy name %s", deploy.Name))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create deploy failed %s", err.Error()))
		}
	}

	deploy.SetID(deploy.Name)
	return deploy, nil
}

func (m *DeploymentManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sDeploys, err := getDeployments(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list deployment info failed:%s", err.Error())
		}
		return nil
	}

	var deploys []*types.Deployment
	for _, ns := range k8sDeploys.Items {
		if deploy, err := k8sDeployToSCDeploy(cluster.KubeClient, &ns); err != nil {
			log.Warnf("list deployment info failed:%s", err.Error())
			return nil
		} else {
			deploys = append(deploys, deploy)
		}
	}
	return deploys
}

func (m *DeploymentManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get deployment info failed:%s", err.Error())
		}
		return nil
	}

	if deploy, err := k8sDeployToSCDeploy(cluster.KubeClient, k8sDeploy); err != nil {
		log.Warnf("get deployment info failed:%s", err.Error())
		return nil
	} else {
		return deploy
	}
}

func (m *DeploymentManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)

	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("deployment %s doesn't exist", namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment failed %s", err.Error()))
		}
	}

	if err := deleteDeployment(cluster.KubeClient, namespace, deploy.GetID()); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete deployment failed %s", err.Error()))
	}

	if delete, ok := k8sDeploy.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cluster.KubeClient, namespace, k8sDeploy.Spec.Template.Spec.Volumes)
	}
	return nil
}

func (m *DeploymentManager) Action(ctx *resource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case types.ActionGetHistory:
		return m.getDeploymentHistory(ctx)
	case types.ActionRollback:
		return nil, m.rollback(ctx)
	case types.ActionSetImage:
		return nil, m.setImage(ctx)
	case types.ActionSetPodCount:
		return m.setPodCount(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func getDeployment(cli client.Client, namespace, name string) (*appsv1.Deployment, error) {
	deploy := appsv1.Deployment{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &deploy)
	return &deploy, err
}

func getDeployments(cli client.Client, namespace string) (*appsv1.DeploymentList, error) {
	deploys := appsv1.DeploymentList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &deploys)
	return &deploys, err
}

func createDeployment(cli client.Client, namespace string, deploy *types.Deployment) error {
	podTemplate, k8sPVCs, err := createPodTempateSpec(namespace, deploy, cli)
	if err != nil {
		return err
	}

	replicas := int32(deploy.Replicas)
	k8sDeploy := &appsv1.Deployment{
		ObjectMeta: generatePodOwnerObjectMeta(namespace, deploy),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploy.Name},
			},
			Template: *podTemplate,
		},
	}

	if err := cli.Create(context.TODO(), k8sDeploy); err != nil {
		deletePVCs(cli, namespace, k8sPVCs)
		return err
	}

	return nil
}

func deleteDeployment(cli client.Client, namespace, name string) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), deploy)
}

func k8sDeployToSCDeploy(cli client.Client, k8sDeploy *appsv1.Deployment) (*types.Deployment, error) {
	containers, templates := k8sPodSpecToScContainersAndVCTemplates(k8sDeploy.Spec.Template.Spec.Containers,
		k8sDeploy.Spec.Template.Spec.Volumes)

	pvs, err := getPVCs(cli, k8sDeploy.Namespace, templates)
	if err != nil {
		return nil, err
	}

	var advancedOpts types.AdvancedOptions
	opts, ok := k8sDeploy.Annotations[AnnkeyForWordloadAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	deploy := &types.Deployment{
		Name:              k8sDeploy.Name,
		Replicas:          int(*k8sDeploy.Spec.Replicas),
		Containers:        containers,
		PersistentVolumes: pvs,
		AdvancedOptions:   advancedOpts,
		Status:            k8sWorkloadStatusToScWorkloadStatus(&k8sDeploy.Status),
	}
	deploy.SetID(k8sDeploy.Name)
	deploy.SetCreationTimestamp(k8sDeploy.CreationTimestamp.Time)
	deploy.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sDeploy.Spec.Template.Annotations)
	return deploy, nil
}

func (m *DeploymentManager) getDeploymentHistory(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	_, replicasets, err := getDeploymentAndReplicaSets(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("deployment %s doesn't exist", namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment history failed %s", err.Error()))
		}
	}

	var versionInfos types.VersionInfos
	for _, rs := range replicasets {
		if v, ok := rs.Annotations[RevisionAnnotation]; ok {
			version, _ := strconv.Atoi(v)
			containers, _ := k8sPodSpecToScContainersAndVCTemplates(rs.Spec.Template.Spec.Containers, nil)
			versionInfos = append(versionInfos, types.VersionInfo{
				Name:         deploy.GetID(),
				Namespace:    namespace,
				Version:      version,
				ChangeReason: rs.Annotations[ChangeCauseAnnotation],
				Containers:   containers,
			})
		}
	}

	sort.Sort(versionInfos)
	return &types.VersionHistory{
		VersionInfos: versionInfos[:len(versionInfos)-1],
	}, nil
}

func (m *DeploymentManager) rollback(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Resource.GetAction().Input.(*types.RollBackVersion)
	if ok == false {
		return resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action rollback version param is not valid"))
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, replicasets, err := getDeploymentAndReplicaSets(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("deployment %s doesn't exist", namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed %s", err.Error()))
		}
	}

	if k8sDeploy.Spec.Paused {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("cannot rollback a paused deployment"))
	}

	var rsForVersion *appsv1.ReplicaSet
	for _, replicaset := range replicasets {
		if v, ok := replicaset.Annotations[RevisionAnnotation]; ok {
			if v == strconv.Itoa(param.Version) {
				rsForVersion = &replicaset
				break
			}
		}
	}

	if rsForVersion == nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed no found version"))
	}

	delete(rsForVersion.Spec.Template.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	annotations := map[string]string{}
	for k := range AnnotationsToSkip {
		if v, ok := k8sDeploy.Annotations[k]; ok {
			annotations[k] = v
		}
	}
	for k, v := range rsForVersion.Annotations {
		if !AnnotationsToSkip[k] {
			annotations[k] = v
		}
	}

	annotations[ChangeCauseAnnotation] = param.Reason
	patch, err := marshalPatch(rsForVersion.Spec.Template, annotations)
	if err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("marshal deployment patch when rollback failed: %v", err.Error()))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sDeploy, k8stypes.JSONPatchType, patch); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed: %v", err.Error()))
	}

	return nil
}

func (m *DeploymentManager) setImage(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Resource.GetAction().Input.(*types.SetImage)
	if ok == false {
		return resterror.NewAPIError(resterror.InvalidFormat, "action set image param is not valid")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("deployment %s doesn't exist", namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment failed %s", err.Error()))
		}
	}

	patch, err := getSetImagePatch(param, k8sDeploy.Spec.Template, k8sDeploy.Annotations)
	if err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("get deployment patch when set image failed: %v", err.Error()))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sDeploy, k8stypes.JSONPatchType, patch); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("set deployment image failed: %v", err.Error()))
	}

	return nil
}

func (m *DeploymentManager) setPodCount(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	param, ok := ctx.Resource.GetAction().Input.(*types.SetPodCount)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, "action set pod count param is not valid")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("deployment %s doesn't exist", deploy.GetID()))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment failed %s", err.Error()))
		}
	}

	if int(*k8sDeploy.Spec.Replicas) == param.Replicas {
		return param, nil
	} else {
		replicas := int32(param.Replicas)
		k8sDeploy.Spec.Replicas = &replicas
		if err := cluster.KubeClient.Update(context.TODO(), k8sDeploy); err != nil {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("set deployment pod count failed %s", err.Error()))
		} else {
			return param, nil
		}
	}
}

func getDeploymentAndReplicaSets(cli client.Client, namespace, deployName string) (*appsv1.Deployment, []appsv1.ReplicaSet, error) {
	k8sDeploy, err := getDeployment(cli, namespace, deployName)
	if err != nil {
		return nil, nil, err
	}

	if k8sDeploy.Spec.Selector == nil {
		return nil, nil, fmt.Errorf("deploy %v has no selector", k8sDeploy.Name)
	}

	replicasets := appsv1.ReplicaSetList{}
	opts := &client.ListOptions{Namespace: namespace}
	labels, err := metav1.LabelSelectorAsSelector(k8sDeploy.Spec.Selector)
	if err != nil {
		return nil, nil, err
	}

	opts.LabelSelector = labels
	if err := cli.List(context.TODO(), opts, &replicasets); err != nil {
		return nil, nil, err
	}

	var replicaSetsByDeployControled []appsv1.ReplicaSet
	for _, item := range replicasets.Items {
		if isControllerBy(item.OwnerReferences, k8sDeploy.UID) {
			replicaSetsByDeployControled = append(replicaSetsByDeployControled, item)
		}
	}

	return k8sDeploy, replicaSetsByDeployControled, nil
}

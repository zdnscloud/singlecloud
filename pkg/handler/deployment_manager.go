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
	"github.com/zdnscloud/gorest"
	resttypes "github.com/zdnscloud/gorest/resource"
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
	api.DefaultHandler
	clusters *ClusterManager
}

func newDeploymentManager(clusters *ClusterManager) *DeploymentManager {
	return &DeploymentManager{clusters: clusters}
}

func (m *DeploymentManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	deploy := ctx.Object.(*types.Deployment)
	if err := createDeployment(cluster.KubeClient, namespace, deploy); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate deploy name %s", deploy.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create deploy failed %s", err.Error()))
		}
	}

	deploy.SetID(deploy.Name)
	return deploy, nil
}

func (m *DeploymentManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
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

func (m *DeploymentManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	deploy := ctx.Object.(*types.Deployment)
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

func (m *DeploymentManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	deploy := ctx.Object.(*types.Deployment)

	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("deployment %s desn't exist", namespace))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment failed %s", err.Error()))
		}
	}

	if int(*k8sDeploy.Spec.Replicas) == deploy.Replicas {
		return deploy, nil
	} else {
		replicas := int32(deploy.Replicas)
		k8sDeploy.Spec.Replicas = &replicas
		newDeploy, err := k8sDeployToSCDeploy(cluster.KubeClient, k8sDeploy)
		if err != nil {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update deployment failed %s", err.Error()))
		}

		if err := cluster.KubeClient.Update(context.TODO(), k8sDeploy); err != nil {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update deployment failed %s", err.Error()))
		} else {
			return newDeploy, nil
		}
	}
}

func (m *DeploymentManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	deploy := ctx.Object.(*types.Deployment)

	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("deployment %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment failed %s", err.Error()))
		}
	}

	if err := deleteDeployment(cluster.KubeClient, namespace, deploy.GetID()); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete deployment failed %s", err.Error()))
	}

	if delete, ok := k8sDeploy.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cluster.KubeClient, namespace, k8sDeploy.Spec.Template.Spec.Volumes)
	}
	return nil
}

func (m *DeploymentManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	switch ctx.Action.Name {
	case types.ActionGetHistory:
		return m.getDeploymentHistory(ctx)
	case types.ActionRollback:
		return nil, m.rollback(ctx)
	case types.ActionSetImage:
		return nil, m.setImage(ctx)
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
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
	}
	deploy.SetID(k8sDeploy.Name)
	deploy.SetType(types.DeploymentType)
	deploy.SetCreationTimestamp(k8sDeploy.CreationTimestamp.Time)
	deploy.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sDeploy.Spec.Template.Annotations)
	return deploy, nil
}

func (m *DeploymentManager) getDeploymentHistory(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	deploy := ctx.Object.(*types.Deployment)
	_, replicasets, err := getDeploymentAndReplicaSets(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("deployment %s desn't exist", namespace))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment history failed %s", err.Error()))
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

func (m *DeploymentManager) rollback(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Action.Input.(*types.RollBackVersion)
	if ok == false || param.Version < 0 {
		return resttypes.NewAPIError(resttypes.InvalidFormat,
			fmt.Sprintf("rollback version param is not valid: %v", ctx.Action.Input))
	}

	namespace := ctx.Object.GetParent().GetID()
	deploy := ctx.Object.(*types.Deployment)
	k8sDeploy, replicasets, err := getDeploymentAndReplicaSets(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("deployment %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed %s", err.Error()))
		}
	}

	if k8sDeploy.Spec.Paused {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("cannot rollback a paused deployment"))
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
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed no found version"))
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
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("marshal deployment patch when rollback failed: %v", err.Error()))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sDeploy, k8stypes.JSONPatchType, patch); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed: %v", err.Error()))
	}

	return nil
}

func (m *DeploymentManager) setImage(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Action.Input.(*types.SetImage)
	if ok == false || len(param.Images) == 0 {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "set image param is not valid")
	}

	namespace := ctx.Object.GetParent().GetID()
	deploy := ctx.Object.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("deployment %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get deployment failed %s", err.Error()))
		}
	}

	patch, err := getSetImagePatch(param, k8sDeploy.Spec.Template, k8sDeploy.Annotations)
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("get deployment patch when set image failed: %v", err.Error()))
	}

	if err := cluster.KubeClient.Patch(context.TODO(), k8sDeploy, k8stypes.JSONPatchType, patch); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("set deployment image failed: %v", err.Error()))
	}

	return nil
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

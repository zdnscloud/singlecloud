package handler

import (
	"context"
	"encoding/json"
	"fmt"

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
	err := createDeployment(cluster.KubeClient, namespace, deploy)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate deploy name %s", deploy.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create deploy failed %s", err.Error()))
		}
	}

	deploy.SetID(deploy.Name)
	if err := createServiceAndIngress(deploy.Containers, deploy.AdvancedOptions, cluster.KubeClient, namespace, deploy.Name, false); err != nil {
		deleteDeployment(cluster.KubeClient, namespace, deploy.Name)
		return nil, err
	}

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

	opts, ok := k8sDeploy.Annotations[AnnkeyForWordloadAdvancedoption]
	if ok {
		deleteServiceAndIngress(cluster.KubeClient, namespace, deploy.GetID(), opts)
	}

	if delete, ok := k8sDeploy.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cluster.KubeClient, namespace, k8sDeploy.Spec.Template.Spec.Volumes)
	}
	return nil
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

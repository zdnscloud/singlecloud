package k8smanager

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type DeploymentManager struct {
	cluster *types.Cluster
}

func newDeploymentManager(cluster *types.Cluster) DeploymentManager {
	return DeploymentManager{cluster: cluster}
}

func (m DeploymentManager) Create(namespace string, deploy *types.Deployment, yamlConf []byte) (interface{}, *resttypes.APIError) {
	err := createDeployment(m.cluster.KubeClient, namespace, deploy)
	if err == nil {
		deploy.SetID(deploy.Name)
		return deploy, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate deploy name %s", deploy.Name))
	} else {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create deploy failed %s", err.Error()))
	}
}

func (m DeploymentManager) List(namespace string) interface{} {
	k8sDeploys, err := getDeployments(m.cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list deployment info failed:%s", err.Error())
		}
		return nil
	}

	var deploys []*types.Deployment
	for _, ns := range k8sDeploys.Items {
		deploys = append(deploys, k8sDeployToSCDeploy(&ns))
	}
	return deploys
}

func (m DeploymentManager) Get(namespace string, deploy *types.Deployment) interface{} {
	k8sDeploy, err := getDeployment(m.cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get deployment info failed:%s", err.Error())
		}
		return nil
	}

	return k8sDeployToSCDeploy(k8sDeploy)
}

func (m DeploymentManager) Delete(namespace string, deploy *types.Deployment) *resttypes.APIError {
	err := deleteDeployment(m.cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("deployment %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete deployment failed %s", err.Error()))
		}
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
	replica := int32(deploy.Replicas)
	var containers []corev1.Container
	for _, c := range deploy.Containers {
		containers = append(containers, corev1.Container{
			Name:    c.Name,
			Image:   c.Image,
			Command: c.Command,
			Args:    c.Args,
		})
	}
	k8sDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: deploy.Name, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploy.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": deploy.Name}},
				Spec:       corev1.PodSpec{Containers: containers},
			},
		},
	}
	return cli.Create(context.TODO(), k8sDeploy)
}

func deleteDeployment(cli client.Client, namespace, name string) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), deploy)
}

func k8sDeployToSCDeploy(k8sDeploy *appsv1.Deployment) *types.Deployment {
	var containers []types.Container
	for _, c := range k8sDeploy.Spec.Template.Spec.Containers {
		containers = append(containers, types.Container{
			Name:    c.Name,
			Image:   c.Image,
			Command: c.Command,
			Args:    c.Args,
		})
	}
	deploy := &types.Deployment{
		Name:       k8sDeploy.Name,
		Replicas:   uint32(*k8sDeploy.Spec.Replicas),
		Containers: containers,
	}
	deploy.SetID(k8sDeploy.Name)
	deploy.SetType(types.DeploymentType)
	deploy.SetCreationTimestamp(k8sDeploy.CreationTimestamp.Time)
	return deploy
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

const (
	AnnkeyForDeploymentAdvancedoption = "zcloud_deployment_advanded_options"
)

type DeploymentManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newDeploymentManager(clusters *ClusterManager) *DeploymentManager {
	return &DeploymentManager{clusters: clusters}
}

func (m *DeploymentManager) Create(obj resttypes.Object, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := obj.GetParent().GetID()
	deploy := obj.(*types.Deployment)
	err := createDeployment(cluster.KubeClient, namespace, deploy)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate deploy name %s", deploy.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create deploy failed %s", err.Error()))
		}
	}

	deploy.SetID(deploy.Name)
	advancedOpts := deploy.AdvancedOptions
	var servicePorts []types.ServicePort
	var rules []types.IngressRule
	for _, s := range advancedOpts.ExposedServices {
		servicePorts = append(servicePorts, types.ServicePort{
			Name:       s.Name,
			Port:       s.ServicePort,
			TargetPort: s.Port,
			Protocol:   s.Protocol,
		})

		if s.AutoCreateIngress {
			rules = append(rules, types.IngressRule{
				Host: s.IngressDomainName,
				Paths: []types.IngressPath{
					types.IngressPath{
						Path:        s.IngressPath,
						ServiceName: deploy.Name,
						ServicePort: s.ServicePort,
					},
				},
			})
		}
	}

	if len(servicePorts) > 0 {
		service := &types.Service{
			Name:         deploy.Name,
			ServiceType:  deploy.AdvancedOptions.ExposedServiceType,
			ExposedPorts: servicePorts,
		}
		err = createService(cluster.KubeClient, namespace, service)
		if err != nil {
			deleteDeployment(cluster.KubeClient, namespace, deploy.Name)
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create service failed %s", err.Error()))
		}

		if len(rules) > 0 {
			ingress := &types.Ingress{
				Name:  deploy.Name,
				Rules: rules,
			}

			err = createIngress(cluster.KubeClient, namespace, ingress)
			if err != nil {
				deleteDeployment(cluster.KubeClient, namespace, deploy.Name)
				deleteService(cluster.KubeClient, namespace, deploy.Name)
				return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create ingress failed %s", err.Error()))
			}
		}
	}

	return deploy, nil
}

func (m *DeploymentManager) List(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	namespace := obj.GetParent().GetID()
	k8sDeploys, err := getDeployments(cluster.KubeClient, namespace)
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

func (m *DeploymentManager) Get(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	namespace := obj.GetParent().GetID()
	deploy := obj.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get deployment info failed:%s", err.Error())
		}
		return nil
	}

	return k8sDeployToSCDeploy(k8sDeploy)
}

func (m *DeploymentManager) Delete(obj resttypes.Object) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := obj.GetParent().GetID()
	deploy := obj.(*types.Deployment)

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

	var advancedOpts types.DeploymentAdvancedOptions
	opts, ok := k8sDeploy.Annotations[AnnkeyForDeploymentAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
		if len(advancedOpts.ExposedServices) > 0 {
			deleteService(cluster.KubeClient, namespace, deploy.GetID())
			for _, s := range advancedOpts.ExposedServices {
				if s.AutoCreateIngress {
					deleteIngress(cluster.KubeClient, namespace, deploy.GetID())
					break
				}
			}
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
	usedConfigMap := make(map[string]struct{})
	for _, c := range deploy.Containers {
		var mounts []corev1.VolumeMount
		var ports []corev1.ContainerPort
		if c.ConfigName != "" {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      c.ConfigName,
				MountPath: c.MountPath,
			})
			usedConfigMap[c.ConfigName] = struct{}{}
		}

		for _, spec := range c.ExposedPorts {
			protocol, err := scProtocolToK8SProtocol(spec.Protocol)
			if err != nil {
				return err
			}
			ports = append(ports, corev1.ContainerPort{
				ContainerPort: int32(spec.Port),
				Protocol:      protocol,
			})
		}

		containers = append(containers, corev1.Container{
			Name:         c.Name,
			Image:        c.Image,
			Command:      c.Command,
			Args:         c.Args,
			VolumeMounts: mounts,
			Ports:        ports,
		})
	}

	var podVolumes []corev1.Volume
	for n, _ := range usedConfigMap {
		configMapSource := &corev1.ConfigMapVolumeSource{}
		configMapSource.Name = n
		source := corev1.VolumeSource{
			ConfigMap: configMapSource,
		}
		podVolumes = append(podVolumes, corev1.Volume{
			Name:         n,
			VolumeSource: source,
		})
	}

	advancedOpts, _ := json.Marshal(deploy.AdvancedOptions)
	k8sDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploy.Name,
			Namespace: namespace,
			Annotations: map[string]string{
				AnnkeyForDeploymentAdvancedoption: string(advancedOpts),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploy.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": deploy.Name}},
				Spec: corev1.PodSpec{
					Containers: containers,
					Volumes:    podVolumes,
				},
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
	containers := K8sContainersToScContainers(k8sDeploy.Spec.Template.Spec.Containers, k8sDeploy.Spec.Template.Spec.Volumes)

	var advancedOpts types.DeploymentAdvancedOptions
	opts, ok := k8sDeploy.Annotations[AnnkeyForDeploymentAdvancedoption]
	if ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	deploy := &types.Deployment{
		Name:            k8sDeploy.Name,
		Replicas:        int(*k8sDeploy.Spec.Replicas),
		Containers:      containers,
		AdvancedOptions: advancedOpts,
	}
	deploy.SetID(k8sDeploy.Name)
	deploy.SetType(types.DeploymentType)
	deploy.SetCreationTimestamp(k8sDeploy.CreationTimestamp.Time)
	return deploy
}

func K8sContainersToScContainers(k8sContainers []corev1.Container, volumes []corev1.Volume) []types.Container {
	var containers []types.Container
	for _, c := range k8sContainers {
		var configName, mountPath string
		for _, vm := range c.VolumeMounts {
			for _, v := range volumes {
				if v.Name == vm.Name && v.ConfigMap != nil {
					configName = v.ConfigMap.Name
					mountPath = vm.MountPath
					break
				}
			}
		}

		var exposedPorts []types.DeploymentPort
		for _, p := range c.Ports {
			exposedPorts = append(exposedPorts, types.DeploymentPort{
				Name:     p.Name,
				Port:     int(p.ContainerPort),
				Protocol: strings.ToLower(string(p.Protocol)),
			})
		}

		containers = append(containers, types.Container{
			Name:         c.Name,
			Image:        c.Image,
			Command:      c.Command,
			Args:         c.Args,
			ConfigName:   configName,
			MountPath:    mountPath,
			ExposedPorts: exposedPorts,
		})
	}

	return containers
}

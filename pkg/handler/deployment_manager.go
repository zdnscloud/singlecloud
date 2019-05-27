package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
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

const (
	AnnkeyForDeploymentAdvancedoption = "zcloud_deployment_advanded_options"
	AnnkeyForPromethusScrape          = "prometheus.io/scrape"
	AnnkeyForPromethusPort            = "prometheus.io/port"
	AnnkeyForPromethusPath            = "prometheus.io/path"
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
	containerPorts := make(map[string]types.ContainerPort)
	for _, container := range deploy.Containers {
		for _, port := range container.ExposedPorts {
			containerPorts[port.Name] = port
		}
	}

	if err := createServiceAndIngress(containerPorts, deploy.AdvancedOptions, cluster.KubeClient, namespace, deploy.Name, false); err != nil {
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
		deploys = append(deploys, k8sDeployToSCDeploy(&ns))
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

	return k8sDeployToSCDeploy(k8sDeploy)
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
		err := cluster.KubeClient.Update(context.TODO(), k8sDeploy)
		if err != nil {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update deployment failed %s", err.Error()))
		} else {
			return k8sDeployToSCDeploy(k8sDeploy), nil
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

	opts, ok := k8sDeploy.Annotations[AnnkeyForDeploymentAdvancedoption]
	if ok {
		deleteServiceAndIngress(cluster.KubeClient, namespace, deploy.GetID(), opts)
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
	replicas := int32(deploy.Replicas)
	k8sPodSpec, err := scContainersToK8sPodSpec(deploy.Containers, nil)
	if err != nil {
		return err
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
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploy.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: scExposedMetricToK8sTempateObjectMeta(deploy.Name, deploy.AdvancedOptions.ExposedMetric),
				Spec:       k8sPodSpec,
			},
		},
	}
	return cli.Create(context.TODO(), k8sDeploy)
}

func scExposedMetricToK8sTempateObjectMeta(name string, exposedMetric types.ExposedMetric) metav1.ObjectMeta {
	templateObjMeta := metav1.ObjectMeta{Labels: map[string]string{"app": name}}
	if exposedMetric.Port != 0 && exposedMetric.Path != "" {
		prometheusConf := make(map[string]string)
		prometheusConf[AnnkeyForPromethusScrape] = "true"
		prometheusConf[AnnkeyForPromethusPort] = strconv.Itoa(exposedMetric.Port)
		prometheusConf[AnnkeyForPromethusPath] = exposedMetric.Path
		templateObjMeta.Annotations = prometheusConf
	}
	return templateObjMeta
}

func deleteDeployment(cli client.Client, namespace, name string) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), deploy)
}

func k8sDeployToSCDeploy(k8sDeploy *appsv1.Deployment) *types.Deployment {
	containers, _ := k8sContainersToScContainersAndPVCTemplate(k8sDeploy.Spec.Template.Spec.Containers,
		k8sDeploy.Spec.Template.Spec.Volumes)

	var advancedOpts types.AdvancedOptions
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
	deploy.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sDeploy.Spec.Template.Annotations)
	return deploy
}

func k8sAnnotationsToScExposedMetric(annotations map[string]string) types.ExposedMetric {
	if doScrape, ok := annotations[AnnkeyForPromethusScrape]; ok && doScrape == "true" {
		port, _ := strconv.Atoi(annotations[AnnkeyForPromethusPort])
		return types.ExposedMetric{
			Port: port,
			Path: annotations[AnnkeyForPromethusPath],
		}
	}
	return types.ExposedMetric{}
}

func k8sContainersToScContainersAndPVCTemplate(k8sContainers []corev1.Container, k8sVolumes []corev1.Volume) ([]types.Container, types.VolumeClaimTemplate) {
	var containers []types.Container
	var template types.VolumeClaimTemplate
	for _, c := range k8sContainers {
		var volumes []types.Volume
		for _, vm := range c.VolumeMounts {
			for _, v := range k8sVolumes {
				if v.Name == vm.Name {
					if v.ConfigMap != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypeConfigMap,
							Name:      v.ConfigMap.Name,
							MountPath: vm.MountPath,
						})
					} else if v.Secret != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypeSecret,
							Name:      v.Secret.SecretName,
							MountPath: vm.MountPath,
						})
					} else if v.PersistentVolumeClaim != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypePersistentVolume,
							Name:      v.PersistentVolumeClaim.ClaimName,
							MountPath: vm.MountPath,
						})
					} else if v.EmptyDir != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypePersistentVolume,
							Name:      v.Name,
							MountPath: vm.MountPath,
						})
						template = types.VolumeClaimTemplate{
							Name:             v.Name,
							StorageClassName: types.StorageClassNameTemp,
						}
						if v.EmptyDir.SizeLimit != nil {
							template.Size = v.EmptyDir.SizeLimit.String()
						}
					}
					break
				}
			}
		}

		var exposedPorts []types.ContainerPort
		for _, p := range c.Ports {
			exposedPorts = append(exposedPorts, types.ContainerPort{
				Name:     p.Name,
				Port:     int(p.ContainerPort),
				Protocol: strings.ToLower(string(p.Protocol)),
			})
		}

		var env []types.EnvVar
		for _, e := range c.Env {
			env = append(env, types.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}

		containers = append(containers, types.Container{
			Name:         c.Name,
			Image:        c.Image,
			Command:      c.Command,
			Args:         c.Args,
			ExposedPorts: exposedPorts,
			Env:          env,
			Volumes:      volumes,
		})
	}

	return containers, template
}

func scContainersToK8sPodSpec(containers []types.Container, emptyDir *corev1.Volume) (corev1.PodSpec, error) {
	var k8sContainers []corev1.Container
	var k8sVolumes []corev1.Volume
	for _, c := range containers {
		var mounts []corev1.VolumeMount
		var ports []corev1.ContainerPort
		var env []corev1.EnvVar
		for _, volume := range c.Volumes {
			readOnly := true
			var volumeSource corev1.VolumeSource
			switch volume.Type {
			case types.VolumeTypeConfigMap:
				volumeSource = corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: volume.Name,
						},
					},
				}
			case types.VolumeTypeSecret:
				volumeSource = corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: volume.Name,
					},
				}
			case types.VolumeTypePersistentVolume:
				readOnly = false
				if emptyDir != nil && volume.Name == emptyDir.Name {
					volumeSource = emptyDir.VolumeSource
				} else {
					volumeSource = corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: volume.Name,
						},
					}
				}
			default:
				return corev1.PodSpec{}, fmt.Errorf("volume type %s is unsupported", volume.Type)
			}

			k8sVolumes = append(k8sVolumes, corev1.Volume{
				Name:         volume.Name,
				VolumeSource: volumeSource,
			})
			mounts = append(mounts, corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: volume.MountPath,
				ReadOnly:  readOnly,
			})
		}

		var portNames []string
		for _, spec := range c.ExposedPorts {
			protocol, err := scProtocolToK8SProtocol(spec.Protocol)
			if err != nil {
				return corev1.PodSpec{}, fmt.Errorf("invalid protocol for container port")
			}

			if spec.Name == "" {
				return corev1.PodSpec{}, fmt.Errorf("exposed port has no name")
			}

			for _, pn := range portNames {
				if pn == spec.Name {
					return corev1.PodSpec{}, fmt.Errorf("duplicate container port name")
				}
			}
			portNames = append(portNames, spec.Name)

			ports = append(ports, corev1.ContainerPort{
				ContainerPort: int32(spec.Port),
				Protocol:      protocol,
			})
		}

		for _, e := range c.Env {
			env = append(env, corev1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}

		k8sContainers = append(k8sContainers, corev1.Container{
			Name:         c.Name,
			Image:        c.Image,
			Command:      c.Command,
			Args:         c.Args,
			VolumeMounts: mounts,
			Ports:        ports,
			Env:          env,
		})
	}

	return corev1.PodSpec{
		Containers: k8sContainers,
		Volumes:    k8sVolumes,
	}, nil
}

func createServiceAndIngress(containerPorts map[string]types.ContainerPort, advancedOpts types.AdvancedOptions, cli client.Client, namespace, serviceName string, headless bool) *resttypes.APIError {
	var servicePorts []types.ServicePort
	var rules []types.IngressRule
	for _, s := range advancedOpts.ExposedServices {
		containerPort, ok := containerPorts[s.ContainerPortName]
		if ok == false {
			return resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("unknown container port with name:%s", s.ContainerPortName))
		}

		servicePorts = append(servicePorts, types.ServicePort{
			Name:       containerPort.Name,
			Port:       s.ServicePort,
			TargetPort: containerPort.Port,
			Protocol:   string(scIngressProtocolToK8SProtocol(s.IngressProtocol)),
		})

		if s.AutoCreateIngress {
			rules = append(rules, types.IngressRule{
				Host:     s.IngressHost,
				Port:     s.IngressPort,
				Protocol: s.IngressProtocol,
				Paths: []types.IngressPath{
					types.IngressPath{
						Path:        s.IngressPath,
						ServiceName: serviceName,
						ServicePort: s.ServicePort,
					},
				},
			})
		}
	}

	if len(servicePorts) > 0 {
		service := &types.Service{
			Name:         serviceName,
			ServiceType:  advancedOpts.ExposedServiceType,
			ExposedPorts: servicePorts,
		}

		if err := createService(cli, namespace, service, headless); err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create service failed %s", err.Error()))
		}

		if len(rules) > 0 {
			ingress := &types.Ingress{
				Name:  serviceName,
				Rules: rules,
			}

			if err := createIngress(cli, namespace, ingress); err != nil {
				deleteService(cli, namespace, serviceName)
				return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create ingress failed %s", err.Error()))
			}
		}
	}

	return nil
}

func deleteServiceAndIngress(cli client.Client, namespace, serviceName, opts string) {
	var advancedOpts types.AdvancedOptions
	json.Unmarshal([]byte(opts), &advancedOpts)
	if len(advancedOpts.ExposedServices) > 0 {
		deleteService(cli, namespace, serviceName)
		for _, s := range advancedOpts.ExposedServices {
			if s.AutoCreateIngress {
				deleteIngress(cli, namespace, serviceName)
				break
			}
		}
	}
}

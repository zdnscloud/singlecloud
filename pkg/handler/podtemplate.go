package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	VolumeNamePrefix                = "vol"
	AnnkeyForWordloadAdvancedoption = "zcloud_workload_advanded_options"
	AnnkeyForPromethusScrape        = "prometheus.io/scrape"
	AnnkeyForPromethusPort          = "prometheus.io/port"
	AnnkeyForPromethusPath          = "prometheus.io/path"
	AnnKeyForReloadWhenConfigChange = "zcloud.cn/update-on-config-change"
	AnnKeyForConfigHashAnnotation   = "zcloud.cn/config-hash"
)

func createPodTempateSpec(namespace string, podOwner interface{}, cli client.Client) (*corev1.PodTemplateSpec, []corev1.PersistentVolumeClaim, error) {
	structVal := reflect.ValueOf(podOwner).Elem()
	advancedOpts := structVal.FieldByName("AdvancedOptions").Interface().(types.AdvancedOptions)
	containers := structVal.FieldByName("Containers").Interface().([]types.Container)
	pvcs := structVal.FieldByName("VolumeClaimTemplates").Interface().([]types.VolumeClaimTemplate)

	k8sPodSpec, k8sPVCs, err := scPodSpecToK8sPodSpecAndPVCs(containers, pvcs)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := podOwner.(*types.StatefulSet); ok == false {
		if err := createPVCs(cli, namespace, k8sPVCs); err != nil {
			return nil, nil, err
		}
	}

	name := structVal.FieldByName("Name").String()
	meta, err := createPodTempateObjectMeta(name, namespace, cli, advancedOpts, containers)
	if err != nil {
		return nil, nil, err
	}

	return &corev1.PodTemplateSpec{
		ObjectMeta: meta,
		Spec:       k8sPodSpec,
	}, k8sPVCs, nil
}

func generatePodOwnerObjectMeta(namespace string, podOwner interface{}) metav1.ObjectMeta {
	structVal := reflect.ValueOf(podOwner).Elem()
	advancedOpts := structVal.FieldByName("AdvancedOptions").Interface().(types.AdvancedOptions)
	opts, _ := json.Marshal(advancedOpts)
	annotations := map[string]string{
		AnnkeyForWordloadAdvancedoption: string(opts),
	}
	if advancedOpts.ReloadWhenConfigChange {
		annotations[AnnKeyForReloadWhenConfigChange] = "true"
	}
	return metav1.ObjectMeta{
		Name:        structVal.FieldByName("Name").String(),
		Namespace:   namespace,
		Annotations: annotations,
	}
}

func createPVCs(cli client.Client, namespace string, k8sPVCs []corev1.PersistentVolumeClaim) error {
	for _, pvc := range k8sPVCs {
		pvc.Namespace = namespace
		if err := cli.Create(context.TODO(), &pvc); err != nil {
			return fmt.Errorf("create pvc %s with namespace %s failed: %s", pvc.Name, namespace, err.Error())
		}
	}

	return nil
}

func getPVCs(cli client.Client, namespace string, templates []types.VolumeClaimTemplate) ([]types.VolumeClaimTemplate, error) {
	var vcTemplates []types.VolumeClaimTemplate
	for _, template := range templates {
		if template.StorageClassName != types.StorageClassNameTemp {
			if k8sPVC, err := getPersistentVolumeClaim(cli, namespace, template.Name); err != nil {
				return nil, err
			} else {
				pvc := k8sPVCToSCPVC(k8sPVC)
				vcTemplates = append(vcTemplates, types.VolumeClaimTemplate{
					Name:             pvc.Name,
					Size:             pvc.RequestStorageSize,
					StorageClassName: pvc.StorageClassName,
				})
			}
		}
	}

	return vcTemplates, nil
}

func scPodSpecToK8sPodSpecAndPVCs(containers []types.Container, pvcs []types.VolumeClaimTemplate) (corev1.PodSpec, []corev1.PersistentVolumeClaim, error) {
	var k8sPodSpec corev1.PodSpec
	k8sEmptyDirs, k8sPVCs, err := scPVCsToK8sVolumesAndPVCs(pvcs)
	if err != nil {
		return k8sPodSpec, nil, err
	}

	k8sPodSpec, err = scContainersAndPVToK8sPodSpec(containers, k8sEmptyDirs, k8sPVCs)
	return k8sPodSpec, k8sPVCs, err
}

func scPVCsToK8sVolumesAndPVCs(pvcs []types.VolumeClaimTemplate) ([]corev1.Volume, []corev1.PersistentVolumeClaim, error) {
	if len(pvcs) == 0 {
		return nil, nil, nil
	}

	var k8sEmptydirVolumes []corev1.Volume
	var k8sPVCs []corev1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		if pvc.StorageClassName == "" {
			return nil, nil, fmt.Errorf("pvc storageclass name should not be empty")
		}

		var k8sQuantity *resource.Quantity
		if pvc.Size != "" {
			quantity, err := resource.ParseQuantity(pvc.Size)
			if err != nil {
				return nil, nil, fmt.Errorf("parse storage size %s failed: %s", pvc.Size, err.Error())
			}
			k8sQuantity = &quantity
		}

		var accessModes []corev1.PersistentVolumeAccessMode
		switch pvc.StorageClassName {
		case types.StorageClassNameTemp:
			k8sEmptydirVolumes = append(k8sEmptydirVolumes, corev1.Volume{
				Name: pvc.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: k8sQuantity,
					},
				},
			})
			continue
		case types.StorageClassNameLVM:
			accessModes = append(accessModes, corev1.ReadWriteOnce)
		case types.StorageClassNameNFS:
			fallthrough
		case types.StorageClassNameCeph:
			accessModes = append(accessModes, corev1.ReadWriteMany)
		default:
			return nil, nil, fmt.Errorf("volumeclaimtemplate storageclass %s isn`t supported", pvc.StorageClassName)
		}

		if k8sQuantity == nil {
			return nil, nil, fmt.Errorf("volumeClaimTemplates storage size must not be zero")
		}

		k8sPVCs = append(k8sPVCs, corev1.PersistentVolumeClaim{
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
	}

	return k8sEmptydirVolumes, k8sPVCs, nil
}

func scContainersAndPVToK8sPodSpec(containers []types.Container, k8sEmptyDirs []corev1.Volume, k8sPVCs []corev1.PersistentVolumeClaim) (corev1.PodSpec, error) {
	var k8sContainers []corev1.Container
	var k8sVolumes []corev1.Volume
	for _, c := range containers {
		var mounts []corev1.VolumeMount
		var ports []corev1.ContainerPort
		var env []corev1.EnvVar
		for i, volume := range c.Volumes {
			readOnly := true
			volumeName := VolumeNamePrefix + strconv.Itoa(i)
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
				found := false
				for _, emptydir := range k8sEmptyDirs {
					if emptydir.Name == volume.Name {
						volumeName = emptydir.Name
						volumeSource = emptydir.VolumeSource
						found = true
						break
					}
				}

				if found == false {
					for _, pvc := range k8sPVCs {
						if pvc.Name == volume.Name {
							volumeName = pvc.Name
							volumeSource = corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: volume.Name,
								},
							}
							found = true
							break
						}
					}
				}

				if found == false {
					return corev1.PodSpec{}, fmt.Errorf("no found volume %s in persistent volume", volume.Name)
				}
			default:
				return corev1.PodSpec{}, fmt.Errorf("volume type %s is unsupported", volume.Type)
			}

			k8sVolumes = append(k8sVolumes, corev1.Volume{
				Name:         volumeName,
				VolumeSource: volumeSource,
			})
			mounts = append(mounts, corev1.VolumeMount{
				Name:      volumeName,
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

func k8sPodSpecToScContainersAndVCTemplates(k8sContainers []corev1.Container, k8sVolumes []corev1.Volume) ([]types.Container, []types.VolumeClaimTemplate) {
	var containers []types.Container
	var templates []types.VolumeClaimTemplate
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
						templates = append(templates, types.VolumeClaimTemplate{
							Name: v.PersistentVolumeClaim.ClaimName,
						})
					} else if v.EmptyDir != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypePersistentVolume,
							Name:      v.Name,
							MountPath: vm.MountPath,
						})
						template := types.VolumeClaimTemplate{
							Name:             v.Name,
							StorageClassName: types.StorageClassNameTemp,
						}
						if v.EmptyDir.SizeLimit != nil {
							template.Size = v.EmptyDir.SizeLimit.String()
						}
						templates = append(templates, template)
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

	return containers, templates
}

func createPodTempateObjectMeta(name, namespace string, cli client.Client, advancedOpts types.AdvancedOptions, containers []types.Container) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Labels:      map[string]string{"app": name},
		Annotations: make(map[string]string)}

	exposedMetric := advancedOpts.ExposedMetric
	if exposedMetric.Port != 0 && exposedMetric.Path != "" {
		meta.Annotations[AnnkeyForPromethusScrape] = "true"
		meta.Annotations[AnnkeyForPromethusPort] = strconv.Itoa(exposedMetric.Port)
		meta.Annotations[AnnkeyForPromethusPath] = exposedMetric.Path
	}

	if advancedOpts.ReloadWhenConfigChange {
		configs, err := getConfigmapAndSecretContainersUse(namespace, cli, containers)
		if err != nil {
			return meta, err
		}

		if len(configs) > 0 {
			hash, err := calculateConfigHash(configs)
			if err != nil {
				return meta, err
			}
			meta.Annotations[AnnKeyForConfigHashAnnotation] = hash
		}
	}

	return meta, nil
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

func createServiceAndIngress(containers []types.Container, advancedOpts types.AdvancedOptions, cli client.Client, namespace, serviceName string, headless bool) *resttypes.APIError {
	containerPorts := make(map[string]types.ContainerPort)
	for _, container := range containers {
		for _, port := range container.ExposedPorts {
			containerPorts[port.Name] = port
		}
	}

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

func getPodOwnerAttributes(podOwner interface{}) (types.AdvancedOptions, []types.Container, []types.VolumeClaimTemplate) {
	opts := reflect.ValueOf(podOwner).Elem().FieldByName("AdvancedOptions").Interface().(types.AdvancedOptions)
	return opts, nil, nil
}

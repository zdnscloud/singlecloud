package handler

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	NoneClusterIP = "None"
)

type ServiceManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newServiceManager(clusters *ClusterManager) *ServiceManager {
	return &ServiceManager{clusters: clusters}
}

func (m *ServiceManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	service := ctx.Object.(*types.Service)
	err := createService(cluster.KubeClient, namespace, service)
	if err == nil {
		service.SetID(service.Name)
		return service, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate service name %s", service.Name))
	} else {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create service failed %s", err.Error()))
	}
}

func (m *ServiceManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sServices, err := getServices(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list service info failed:%s", err.Error())
		}
		return nil
	}

	var services []*types.Service
	for _, sv := range k8sServices.Items {
		services = append(services, k8sServiceToSCService(&sv))
	}
	return services
}

func (m *ServiceManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	service := ctx.Object.(*types.Service)
	k8sService, err := getService(cluster.KubeClient, namespace, service.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get service info failed:%s", err.Error())
		}
		return nil
	}

	return k8sServiceToSCService(k8sService)
}

func (m *ServiceManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	service := ctx.Object.(*types.Service)
	err := deleteService(cluster.KubeClient, namespace, service.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("service %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete service failed %s", err.Error()))
		}
	}
	return nil
}

func getService(cli client.Client, namespace, name string) (*corev1.Service, error) {
	service := corev1.Service{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &service)
	return &service, err
}

func getServices(cli client.Client, namespace string) (*corev1.ServiceList, error) {
	services := corev1.ServiceList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &services)
	return &services, err
}

func createService(cli client.Client, namespace string, service *types.Service) error {
	typ, err := scServiceTypeToK8sServiceType(service.ServiceType)
	if err != nil {
		return err
	}

	var ports []corev1.ServicePort
	for _, p := range service.ExposedPorts {
		protocol, err := scProtocolToK8SProtocol(p.Protocol)
		if err != nil {
			return err
		}

		ports = append(ports, corev1.ServicePort{
			Name:       p.Name,
			Protocol:   protocol,
			Port:       int32(p.Port),
			TargetPort: intstr.FromInt(p.TargetPort),
		})
	}

	k8sService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: service.Name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": service.Name},
			Ports:    ports,
			Type:     typ,
		},
	}

	if typ == corev1.ServiceTypeClusterIP && service.Headless {
		k8sService.Spec.ClusterIP = NoneClusterIP
	}
	return cli.Create(context.TODO(), k8sService)
}

func deleteService(cli client.Client, namespace, name string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), service)
}

func k8sServiceToSCService(k8sService *corev1.Service) *types.Service {
	var ports []types.ServicePort
	for _, p := range k8sService.Spec.Ports {
		ports = append(ports, types.ServicePort{
			Name:       p.Name,
			Protocol:   strings.ToLower(string(p.Protocol)),
			Port:       int(p.Port),
			TargetPort: p.TargetPort.IntValue(),
		})
	}
	service := &types.Service{
		Name:         k8sService.Name,
		ServiceType:  strings.ToLower(string(k8sService.Spec.Type)),
		ExposedPorts: ports,
	}
	service.SetID(k8sService.Name)
	service.SetType(types.ServiceType)
	service.SetCreationTimestamp(k8sService.CreationTimestamp.Time)
	return service
}

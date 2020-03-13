package handler

import (
	"fmt"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	NginxIngressNamespace = "ingress-nginx"
	NginxUDPConfigMapName = "udp-services"
	NginxTCPConfigMapName = "tcp-services"

	annNginxIngressClassKey   = "kubernetes.io/ingress.class"
	annNginxIngressClassValue = "nginx"
)

type UDPIngressManager struct {
	clusters *ClusterManager
}

func newUDPIngressManager(clusters *ClusterManager) *UDPIngressManager {
	mgr := &UDPIngressManager{clusters: clusters}
	go mgr.eventLoop()
	return mgr
}

func (m *UDPIngressManager) eventLoop() {
	eventCh := eventbus.SubscribeResourceEvent(
		types.Namespace{},
		types.Service{})
	for {
		event := <-eventCh
		switch e := event.(type) {
		case eventbus.ResourceDeleteEvent:
			switch r := e.Resource.(type) {
			case *types.Namespace:
				cluster := m.clusters.GetClusterForSubResource(r)
				if cluster != nil {
					if err := clearTransportLayerIngress(cluster.GetKubeClient(), r.GetID(), ""); err != nil {
						log.Warnf("clean udp ingress for namespace %s failed:%s", r.GetID(), err.Error())
					}
				}
			case *types.Service:
				cluster := m.clusters.GetClusterForSubResource(r)
				if cluster != nil {
					namespace := r.GetParent().GetID()
					if err := clearTransportLayerIngress(cluster.GetKubeClient(), namespace, r.GetID()); err != nil {
						log.Warnf("delete udp ingress for svc %s failed:%s", r.GetID(), err.Error())
					}
				}
			}
		}
	}
}

func (m *UDPIngressManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	ingress := ctx.Resource.(*types.UDPIngress)
	err := createUDPIngress(cluster.GetKubeClient(), namespace, ingress)
	if err == nil {
		ingress.SetID(strconv.Itoa(ingress.Port))
		return ingress, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate ingress name %s", ingress.ServiceName))
	} else {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create ingress failed %s", err.Error()))
	}
}

func (m *UDPIngressManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}
	namespace := ctx.Resource.GetParent().GetID()
	ingresses, err := getTransportLayerIngress(cluster.GetKubeClient(), namespace, "")
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list udp ingresses failed:%s", err.Error()))
	}
	return ingresses, nil
}

func (m *UDPIngressManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	udpIngressName := ctx.Resource.GetID()
	udpRules, err := getTransportLayerIngress(cluster.GetKubeClient(), namespace, udpIngressName)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get udp ingress failed: %s", err.Error()))
	} else if len(udpRules) == 1 {
		return udpRules[0], nil
	} else {
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found udp ingress %s", udpIngressName))
	}
}

func (m *UDPIngressManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	hasIngress, err := deleteTransportLayerIngress(cluster.GetKubeClient(), ctx.Resource.GetID())
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete ingress failed %s", err.Error()))
	} else if hasIngress == false {
		return resterror.NewAPIError(resterror.NotFound, "udp ingress doesn't exist")
	} else {
		return nil
	}
}

func deleteTransportLayerIngress(cli client.Client, port string) (bool, error) {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUDPConfigMapName)
	if err != nil {
		return false, err
	}

	cm := k8sConfigMapToSCConfigMap(k8sCM)
	for i, c := range cm.Configs {
		if c.Name == port {
			cm.Configs = append(cm.Configs[:i], cm.Configs[i+1:]...)
			return true, updateConfigMap(cli, NginxIngressNamespace, cm)
		}
	}
	return false, nil
}

func clearTransportLayerIngress(cli client.Client, namespace, svcName string) error {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUDPConfigMapName)
	if err != nil {
		return err
	}

	prefix := namespace + "/"
	if svcName != "" {
		prefix = prefix + svcName + ":"
	}
	cm := k8sConfigMapToSCConfigMap(k8sCM)
	var newConfigs []types.Config
	for _, c := range cm.Configs {
		if strings.HasPrefix(c.Data, prefix) == false {
			newConfigs = append(newConfigs, c)
		}
	}

	if len(newConfigs) != len(cm.Configs) {
		cm.Configs = newConfigs
		return updateConfigMap(cli, NginxIngressNamespace, cm)
	} else {
		return nil
	}
}

func getTransportLayerIngress(cli client.Client, namespace, portStr string) ([]*types.UDPIngress, error) {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUDPConfigMapName)
	if err != nil {
		return nil, err
	}

	cm := k8sConfigMapToSCConfigMap(k8sCM)
	var ingresses []*types.UDPIngress
	for _, c := range cm.Configs {
		serviceAndPort := strings.Split(c.Data, ":")
		if len(serviceAndPort) != 2 {
			return nil, fmt.Errorf("nginx config map %s has invalid ingress data %s", NginxUDPConfigMapName, c.Data)
		}

		port, err := strconv.Atoi(c.Name)
		if err != nil || port == 0 {
			return nil, fmt.Errorf("nginx config map %s has invalid ingress port %s", NginxUDPConfigMapName, c.Name)
		}

		namespaceAndService := strings.Split(serviceAndPort[0], "/")
		if len(namespaceAndService) != 2 {
			return nil, fmt.Errorf("nginx config map %s has invalid service format %s", NginxUDPConfigMapName, serviceAndPort[0])
		}

		if namespace != "" {
			if namespaceAndService[0] != namespace {
				continue
			}
		}

		svcPort, err := strconv.Atoi(serviceAndPort[1])
		if err != nil || svcPort == 0 {
			return nil, fmt.Errorf("nginx config map %s has invalid service port %s", NginxUDPConfigMapName, c.Name)
		}

		if portStr != "" && c.Name != portStr {
			continue
		}

		udpIngress := &types.UDPIngress{
			Port:        port,
			ServiceName: namespaceAndService[1],
			ServicePort: svcPort,
		}
		udpIngress.SetID(c.Name)
		ingresses = append(ingresses, udpIngress)

		if portStr != "" {
			break
		}
	}
	return ingresses, nil
}

func createUDPIngress(cli client.Client, namespace string, ingress *types.UDPIngress) error {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUDPConfigMapName)
	if err != nil {
		return err
	}

	cm := k8sConfigMapToSCConfigMap(k8sCM)
	for _, c := range cm.Configs {
		p, err := strconv.Atoi(c.Name)
		if err != nil {
			return fmt.Errorf("nginx config map %s has invalid port %s", NginxUDPConfigMapName, c.Name)
		}

		if p == ingress.Port {
			return fmt.Errorf("port %d is already used in config map %s", p, NginxUDPConfigMapName)
		}
	}

	service := fmt.Sprintf("%s/%s:%d", namespace, ingress.ServiceName, ingress.ServicePort)
	cm.Configs = append(cm.Configs, types.Config{
		Name: strconv.Itoa(ingress.Port),
		Data: service,
	})
	return updateConfigMap(cli, NginxIngressNamespace, cm)
}

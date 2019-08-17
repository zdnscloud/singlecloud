package handler

import (
	"fmt"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
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
	api.DefaultHandler
	clusters *ClusterManager
}

func newUDPIngressManager(clusters *ClusterManager) *UDPIngressManager {
	return &UDPIngressManager{clusters: clusters}
}

func (m *UDPIngressManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	ingress := ctx.Object.(*types.UdpIngress)
	err := createUDPIngress(cluster.KubeClient, namespace, ingress)
	if err == nil {
		ingress.SetID(ingress.ServiceName)
		return ingress, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate ingress name %s", ingress.ServiceName))
	} else {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create ingress failed %s", err.Error()))
	}
}

func (m *UDPIngressManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	ingresses, err := getTransportLayerIngress(cluster.KubeClient, namespace, "")
	if err != nil {
		log.Warnf("get udp ingress failed %s", err.Error())
	}
	return ingresses
}

func (m *UDPIngressManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	udpRules, err := getTransportLayerIngress(cluster.KubeClient, namespace, ctx.Object.GetID())
	if err != nil {
		log.Warnf("get udp ingress failed %s", err.Error())
		return nil
	} else if len(udpRules) == 1 {
		return udpRules[0]
	} else {
		return nil
	}
}

func (m *UDPIngressManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	hasIngress, err := deleteTransportLayerIngress(cluster.KubeClient, namespace, ctx.Object.GetID())
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete ingress failed %s", err.Error()))
	} else if hasIngress == false {
		return resttypes.NewAPIError(resttypes.NotFound, "udp ingress doesn't exist")
	} else {
		return nil
	}
}

func deleteTransportLayerIngress(cli client.Client, namespace, name string) (bool, error) {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUDPConfigMapName)
	if err != nil {
		return false, err
	}

	svcName := fmt.Sprintf("%s/%s", namespace, name)
	cm := k8sConfigMapToSCConfigMap(k8sCM)
	for i, c := range cm.Configs {
		serviceAndPort := strings.Split(c.Data, ":")
		if len(serviceAndPort) == 2 && serviceAndPort[0] == svcName {
			cm.Configs = append(cm.Configs[:i], cm.Configs[i+1:]...)
			return true, updateConfigMap(cli, NginxIngressNamespace, cm)
		}
	}
	return false, nil
}

func clearTransportLayerIngress(cli client.Client, namespace string) error {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUDPConfigMapName)
	if err != nil {
		return err
	}

	prefix := namespace + "/"
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

func getTransportLayerIngress(cli client.Client, namespace, name string) ([]*types.UdpIngress, error) {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUDPConfigMapName)
	if err != nil {
		return nil, err
	}

	svcName := fmt.Sprintf("%s/%s", namespace, name)
	cm := k8sConfigMapToSCConfigMap(k8sCM)
	var ingresses []*types.UdpIngress
	for _, c := range cm.Configs {
		serviceAndPort := strings.Split(c.Data, ":")
		if len(serviceAndPort) == 2 && strings.HasPrefix(serviceAndPort[0], svcName) {
			port, err := strconv.Atoi(c.Name)
			if err != nil || port == 0 {
				return nil, fmt.Errorf("nginx config map %s has invalid ingress port %s", NginxUDPConfigMapName, c.Name)
			}

			svcPort, err := strconv.Atoi(serviceAndPort[1])
			if err != nil || svcPort == 0 {
				return nil, fmt.Errorf("nginx config map %s has invalid service port %s", NginxUDPConfigMapName, c.Name)
			}

			udpIngress := &types.UdpIngress{
				Port:        port,
				ServiceName: name,
				ServicePort: svcPort,
			}
			udpIngress.SetID(name)
			ingresses = append(ingresses, udpIngress)
		}
	}
	return ingresses, nil
}

func createUDPIngress(cli client.Client, namespace string, ingress *types.UdpIngress) error {
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

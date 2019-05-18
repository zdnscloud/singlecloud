package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	extv1beta1 "k8s.io/api/extensions/v1beta1"
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
	NginxIngressNamespace = "ingress-nginx"
	NginxUDPConfigMapName = "udp-services"
	NginxTCPConfigMapName = "tcp-services"

	annNginxIngressClassKey        = "kubernetes.io/ingress.class"
	annNginxIngressClassValue      = "nginx"
	annNginxIngressBackendProtocol = "nginx.ingress.kubernetes.io/backend-protocol"
	annNginxServiceGRPCBackend     = "GRPC"
)

type IngressManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newIngressManager(clusters *ClusterManager) *IngressManager {
	return &IngressManager{clusters: clusters}
}

func (m *IngressManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	ingress := ctx.Object.(*types.Ingress)
	err := createIngress(cluster.KubeClient, namespace, ingress)
	if err == nil {
		ingress.SetID(ingress.Name)
		return ingress, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate ingress name %s", ingress.Name))
	} else {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create ingress failed %s", err.Error()))
	}
}

func (m *IngressManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sIngresss, err := getIngresss(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list ingress info failed:%s", err.Error())
		}
		return nil
	}

	var ingresss []*types.Ingress
	for _, sv := range k8sIngresss.Items {
		ingresss = append(ingresss, k8sIngressToSCIngress(&sv))
	}
	return ingresss
}

func (m *IngressManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	ingress := ctx.Object.(*types.Ingress)
	k8sIngress, err := getIngress(cluster.KubeClient, namespace, ingress.GetID())
	if err == nil {
		ingress = k8sIngressToSCIngress(k8sIngress)
	} else {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get ingress failed %s", err.Error())
			return nil
		}
		ingress.SetID(ingress.GetID())
		ingress.SetType(types.IngressType)
	}

	udpRule, err := getTransportLayerIngress(cluster.KubeClient, namespace, ingress.GetID(), types.IngressProtocolUDP)
	if err != nil {
		log.Warnf("get udp ingress failed %s", err.Error())
		return nil
	}
	if udpRule != nil {
		ingress.Rules = append(ingress.Rules, *udpRule)
	}

	tcpRule, err := getTransportLayerIngress(cluster.KubeClient, namespace, ingress.GetID(), types.IngressProtocolTCP)
	if err != nil {
		log.Warnf("get tcp ingress failed %s", err.Error())
		return nil
	}
	if tcpRule != nil {
		ingress.Rules = append(ingress.Rules, *tcpRule)
	}

	return ingress
}

func (m *IngressManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	hasIngress, err := deleteIngress(cluster.KubeClient, namespace, ctx.Object.GetID())
	if err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete ingress failed %s", err.Error()))
	} else if hasIngress == false {
		return resttypes.NewAPIError(resttypes.NotFound, "ingress doesn't exist")
	} else {
		return nil
	}
}

func getIngress(cli client.Client, namespace, name string) (*extv1beta1.Ingress, error) {
	ingress := extv1beta1.Ingress{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &ingress)
	return &ingress, err
}

func getIngresss(cli client.Client, namespace string) (*extv1beta1.IngressList, error) {
	ingresss := extv1beta1.IngressList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &ingresss)
	return &ingresss, err
}

func createIngress(cli client.Client, namespace string, ingress *types.Ingress) error {
	rules := ingress.Rules
	for _, r := range rules {
		if err := validateRule(&r); err != nil {
			return err
		}
	}
	rules, err := mergeIngressRules(rules)
	if err != nil {
		return err
	}
	var httpRules []extv1beta1.IngressRule
	hasGrpcService := false
	udpServices := make(map[int]string)
	tcpServices := make(map[int]string)
	for _, r := range rules {
		switch r.Protocol {
		case types.IngressProtocolGRPC:
			hasGrpcService = true
			fallthrough
		case types.IngressProtocolHTTP:
			if len(r.Paths) == 0 {
				return fmt.Errorf("invalid ingress with empty path")
			}

			if r.Host == "" {
				return fmt.Errorf("invalid ingress with empty host")
			}

			var paths []extv1beta1.HTTPIngressPath
			for _, p := range r.Paths {
				paths = append(paths, extv1beta1.HTTPIngressPath{
					Path: p.Path,
					Backend: extv1beta1.IngressBackend{
						ServiceName: p.ServiceName,
						ServicePort: intstr.FromInt(p.ServicePort),
					},
				})
			}

			httpRules = append(httpRules, extv1beta1.IngressRule{
				Host: r.Host,
				IngressRuleValue: extv1beta1.IngressRuleValue{
					HTTP: &extv1beta1.HTTPIngressRuleValue{
						Paths: paths,
					},
				},
			})
		case types.IngressProtocolUDP:
			if len(r.Paths) != 1 {
				return fmt.Errorf("for udp protocol, one port can only map to one service")
			}
			if r.Port == 0 {
				return fmt.Errorf("udp ingress port cann't be zero")
			}

			udpServices[r.Port] = fmt.Sprintf("%s/%s:%d", namespace, r.Paths[0].ServiceName, r.Paths[0].ServicePort)

		case types.IngressProtocolTCP:
			if len(r.Paths) != 1 {
				return fmt.Errorf("for tcp protocol, one port can only map to one service")
			}

			if r.Port == 0 {
				return fmt.Errorf("tcp ingress port cann't be zero")
			}

			tcpServices[r.Port] = fmt.Sprintf("%s/%s:%d", namespace, r.Paths[0].ServiceName, r.Paths[0].ServicePort)
		}
	}

	if len(httpRules) > 0 {
		annaotations := map[string]string{
			annNginxIngressClassKey: annNginxIngressClassValue,
		}
		if hasGrpcService {
			annaotations[annNginxIngressBackendProtocol] = annNginxServiceGRPCBackend
		}

		k8sIngress := &extv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:        ingress.Name,
				Namespace:   namespace,
				Annotations: annaotations,
			},
		}
		k8sIngress.Spec = extv1beta1.IngressSpec{
			Rules: httpRules,
		}

		if err := cli.Create(context.TODO(), k8sIngress); err != nil {
			return err
		}
	}

	if len(udpServices) > 0 {
		if err := createTransportLayerIngress(cli, udpServices, types.IngressProtocolUDP); err != nil {
			return err
		}
	}

	if len(tcpServices) > 0 {
		if err := createTransportLayerIngress(cli, tcpServices, types.IngressProtocolTCP); err != nil {
			return err
		}
	}

	return nil
}

func deleteIngress(cli client.Client, namespace, name string) (bool, error) {
	hasHTTPIngress := false
	err := deleteHTTPIngress(cli, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return false, err
		}
	} else {
		hasHTTPIngress = true
	}

	hasUdpIngress, err := deleteTransportLayerIngress(cli, namespace, name, types.IngressProtocolUDP)
	if err != nil {
		return false, err
	}

	hasTCPIngress, err := deleteTransportLayerIngress(cli, namespace, name, types.IngressProtocolTCP)
	if err != nil {
		return false, err
	}

	return hasHTTPIngress || hasUdpIngress || hasTCPIngress, nil
}

func deleteTransportLayerIngress(cli client.Client, namespace, name string, protocol types.IngressProtocol) (bool, error) {
	configMapName := configMapForTransportProtocol(protocol)
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, configMapName)
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

func getTransportLayerIngress(cli client.Client, namespace, name string, protocol types.IngressProtocol) (*types.IngressRule, error) {
	configMapName := configMapForTransportProtocol(protocol)
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, configMapName)
	if err != nil {
		return nil, err
	}

	svcName := fmt.Sprintf("%s/%s", namespace, name)
	cm := k8sConfigMapToSCConfigMap(k8sCM)
	for _, c := range cm.Configs {
		serviceAndPort := strings.Split(c.Data, ":")
		if len(serviceAndPort) == 2 && serviceAndPort[0] == svcName {
			port, err := strconv.Atoi(c.Name)
			if err != nil || port == 0 {
				return nil, fmt.Errorf("nginx config map %s has invalid ingress port %s", configMapName, c.Name)
			}

			svcPort, err := strconv.Atoi(serviceAndPort[1])
			if err != nil || svcPort == 0 {
				return nil, fmt.Errorf("nginx config map %s has invalid service port %s", configMapName, c.Name)
			}

			return &types.IngressRule{
				Port:     port,
				Protocol: protocol,
				Paths: []types.IngressPath{
					types.IngressPath{
						ServiceName: name,
						ServicePort: svcPort,
					},
				},
			}, nil

		}
	}

	return nil, nil
}

func createTransportLayerIngress(cli client.Client, services map[int]string, protocol types.IngressProtocol) error {
	configMapName := configMapForTransportProtocol(protocol)
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, configMapName)
	if err != nil {
		return err
	}

	cm := k8sConfigMapToSCConfigMap(k8sCM)
	for _, c := range cm.Configs {
		p, err := strconv.Atoi(c.Name)
		if err != nil {
			return fmt.Errorf("nginx config map %s has invalid port %s", configMapName, c.Name)
		}

		if _, ok := services[p]; ok {
			return fmt.Errorf("port %d is already used in config map %s", p, configMapName)
		}
	}

	for p, s := range services {
		cm.Configs = append(cm.Configs, types.Config{
			Name: strconv.Itoa(p),
			Data: s,
		})
	}

	return updateConfigMap(cli, NginxIngressNamespace, cm)
}

func configMapForTransportProtocol(protocol types.IngressProtocol) string {
	switch protocol {
	case types.IngressProtocolUDP:
		return NginxUDPConfigMapName
	case types.IngressProtocolTCP:
		return NginxTCPConfigMapName
	default:
		panic("pass invalid protocol")
	}
}

func deleteHTTPIngress(cli client.Client, namespace, name string) error {
	ingress := &extv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), ingress)
}

func k8sIngressToSCIngress(k8sIngress *extv1beta1.Ingress) *types.Ingress {
	rpcBackend, ok := k8sIngress.Annotations[annNginxIngressBackendProtocol]
	isGRPC := ok && rpcBackend == annNginxServiceGRPCBackend
	protocol := types.IngressProtocolHTTP
	if isGRPC {
		protocol = types.IngressProtocolGRPC
	}

	var rules []types.IngressRule
	for _, r := range k8sIngress.Spec.Rules {
		var paths []types.IngressPath
		for _, p := range r.IngressRuleValue.HTTP.Paths {
			paths = append(paths, types.IngressPath{
				Path:        p.Path,
				ServiceName: p.Backend.ServiceName,
				ServicePort: p.Backend.ServicePort.IntValue(),
			})
		}

		rules = append(rules, types.IngressRule{
			Host:     r.Host,
			Paths:    paths,
			Protocol: protocol,
		})
	}

	ingress := &types.Ingress{
		Name:  k8sIngress.Name,
		Rules: rules,
	}
	ingress.SetID(k8sIngress.Name)
	ingress.SetType(types.IngressType)
	ingress.SetCreationTimestamp(k8sIngress.CreationTimestamp.Time)
	return ingress
}

func mergeIngressRules(rules []types.IngressRule) ([]types.IngressRule, error) {
	if len(rules) < 2 {
		return rules, nil
	}

	mergedRules := []types.IngressRule{rules[0]}
	var merged bool
	var err error
	for i := 1; i < len(rules); i++ {
		for j := 0; j < len(mergedRules); j++ {
			merged, err = mergeIngressRule(&rules[i], &mergedRules[j])
			if err != nil {
				return nil, err
			}
			if merged {
				break
			}
		}
		if merged == false {
			mergedRules = append(mergedRules, rules[i])
		}
	}
	return mergedRules, nil
}

func mergeIngressRule(a, b *types.IngressRule) (bool, error) {
	if a.Protocol != b.Protocol {
		return false, nil
	}

	switch a.Protocol {
	case types.IngressProtocolHTTP, types.IngressProtocolGRPC:
		if a.Host != b.Host {
			return false, nil
		}
		for _, ra := range a.Paths {
			for _, rb := range b.Paths {
				if ra.Path == rb.Path {
					return false, fmt.Errorf("duplicate path %s:%s", a.Host, ra.Path)
				}
			}
		}
		b.Paths = append(b.Paths, a.Paths...)
		return true, nil
	case types.IngressProtocolTCP, types.IngressProtocolUDP:
		if a.Port != b.Port {
			return false, nil
		} else {
			return false, fmt.Errorf("duplicate port number %d", a.Port)
		}
		panic("unknown protocol:" + a.Protocol)
	}

	return false, nil
}

func validateRule(r *types.IngressRule) error {
	switch r.Protocol {
	case types.IngressProtocolHTTP, types.IngressProtocolGRPC:
		if r.Host == "" {
			return fmt.Errorf("http ingress should have host")
		}

		if r.Port != 0 {
			return fmt.Errorf("http ingress shouldn't specify port")
		}
	case types.IngressProtocolTCP, types.IngressProtocolUDP:
		if r.Port == 0 {
			return fmt.Errorf("udp/tcp ingress should has nonzero port")
		}
		if r.Host != "" {
			return fmt.Errorf("udp/tcp ingress shouldn't have host")
		}

		if len(r.Paths) != 1 || r.Paths[0].Path != "" {
			return fmt.Errorf("for udp/tcp ingress should have no path and only one backend service")
		}
	default:
		return fmt.Errorf("unsupported ingress protocol:%s", r.Protocol)
	}

	if len(r.Paths) == 0 {
		return fmt.Errorf("ingress with empty path")
	}

	knownPaths := make(map[string]struct{})
	for _, path := range r.Paths {
		if path.ServiceName == "" || path.ServicePort == 0 {
			return fmt.Errorf("service name or port shouldn't empty")
		}
		if _, ok := knownPaths[path.Path]; ok {
			return fmt.Errorf("duplicate path:%s", path.Path)
		}
		knownPaths[path.Path] = struct{}{}
	}

	return nil
}

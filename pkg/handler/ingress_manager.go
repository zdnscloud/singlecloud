package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	NginxIngressNamespace = "ingress-nginx"
	NginxUdpConfigMapName = "udp-services"
	AnnkeyForUDPIngress   = "zcloud_ingress_udp"
)

var defaultIngressClassAnnotation = map[string]string{
	"kubernetes.io/ingress.class": "nginx",
}

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
			logger.Warn("list ingress info failed:%s", err.Error())
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
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get ingress info failed:%s", err.Error())
		}
		return nil
	}

	return k8sIngressToSCIngress(k8sIngress)
}

func (m *IngressManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	return deleteIngress(cluster.KubeClient, namespace, ctx.Object.GetID())
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
	var rules []extv1beta1.IngressRule
	udpServices := make(map[int]string)
	var udpRules []types.IngressRule
	for _, r := range ingress.Rules {
		protocol, err := scProtocolToK8SProtocol(r.Protocol)
		if err != nil {
			return err
		}

		if len(r.Paths) == 0 {
			return fmt.Errorf("invalid ingress with empty path")
		}

		if protocol == corev1.ProtocolUDP {
			if len(r.Paths) != 1 {
				return fmt.Errorf("for udp protocol, one port can only map to one service")
			}
			udpServices[r.Port] = fmt.Sprintf("%s/%s:%d", namespace, r.Paths[0].ServiceName, r.Paths[0].ServicePort)
			udpRules = append(udpRules, r)
			continue
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
		rules = append(rules, extv1beta1.IngressRule{
			Host: r.Host,
			IngressRuleValue: extv1beta1.IngressRuleValue{
				HTTP: &extv1beta1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		})
	}

	if len(rules) > 0 || len(udpRules) > 0 {
		annaotations := make(map[string]string)
		for k, v := range defaultIngressClassAnnotation {
			annaotations[k] = v
		}

		if len(udpRules) > 0 {
			udpRulesJson, _ := json.Marshal(udpRules)
			annaotations[AnnkeyForUDPIngress] = string(udpRulesJson)
		}

		k8sIngress := &extv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:        ingress.Name,
				Namespace:   namespace,
				Annotations: annaotations,
			},
		}

		if len(rules) > 0 {
			k8sIngress.Spec = extv1beta1.IngressSpec{
				Rules: rules,
			}
		} else {
			k8sIngress.Spec = extv1beta1.IngressSpec{
				Backend: &extv1beta1.IngressBackend{
					ServiceName: ingress.Rules[0].Paths[0].ServiceName,
					ServicePort: intstr.FromInt(ingress.Rules[0].Paths[0].ServicePort),
				},
			}
		}

		if err := cli.Create(context.TODO(), k8sIngress); err != nil {
			return err
		}
	}

	if len(udpServices) > 0 {
		return createUdpIngresses(cli, udpServices)
	}

	return nil
}

func deleteIngress(cli client.Client, namespace, name string) *resttypes.APIError {
	k8sIngress, err := getIngress(cli, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("ingress %s in namespace %s desn't exist", name, namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get ingress failed %s", err.Error()))
		}
	}

	err = deleteHTTPIngress(cli, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("ingress %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete ingress failed %s", err.Error()))
		}
	}

	udpRulesJson, ok := k8sIngress.Annotations[AnnkeyForUDPIngress]
	if ok {
		var udpRules []types.IngressRule
		json.Unmarshal([]byte(udpRulesJson), &udpRules)

		var udpPorts []int
		for _, r := range udpRules {
			udpPorts = append(udpPorts, r.Port)
		}

		err := deleteUDPIngress(cli, udpPorts)
		if err != nil {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete udp ingress failed %s", err.Error()))
		}
	}

	return nil
}

func createUdpIngresses(cli client.Client, udpServices map[int]string) error {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUdpConfigMapName)
	if err != nil {
		return err
	}

	cm := k8sConfigMapToSCConfigMap(k8sCM)
	for _, c := range cm.Configs {
		p, err := strconv.Atoi(c.Name)
		if err != nil {
			return fmt.Errorf("nginx udp config map has invalid port %s", c.Name)
		}

		if _, ok := udpServices[p]; ok {
			return fmt.Errorf("udp port %d is already used", p)
		}
	}

	for p, s := range udpServices {
		cm.Configs = append(cm.Configs, types.Config{
			Name: strconv.Itoa(p),
			Data: s,
		})
	}

	return updateConfigMap(cli, NginxIngressNamespace, cm)
}

func deleteHTTPIngress(cli client.Client, namespace, name string) error {
	ingress := &extv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), ingress)
}

func deleteUDPIngress(cli client.Client, udpPorts []int) error {
	k8sCM, err := getConfigMap(cli, NginxIngressNamespace, NginxUdpConfigMapName)
	if err != nil {
		return err
	}
	for _, p := range udpPorts {
		delete(k8sCM.Data, strconv.Itoa(p))
	}
	return cli.Update(context.TODO(), k8sCM)
}

func k8sIngressToSCIngress(k8sIngress *extv1beta1.Ingress) *types.Ingress {
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
			Host:  r.Host,
			Paths: paths,
		})
	}

	udpRulesJson, ok := k8sIngress.Annotations[AnnkeyForUDPIngress]
	if ok {
		var udpRules []types.IngressRule
		json.Unmarshal([]byte(udpRulesJson), &udpRules)
		rules = append(rules, udpRules...)
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

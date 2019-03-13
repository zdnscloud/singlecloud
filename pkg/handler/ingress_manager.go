package handler

import (
	"context"
	"fmt"

	extv1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var defaultIngressClassAnnotation = map[string]string{
	"kubernetes.io/ingress.class": "nginx",
}

type IngressManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newIngressManager(clusters *ClusterManager) *IngressManager {
	return &IngressManager{clusters: clusters}
}

func (m *IngressManager) Create(obj resttypes.Object, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := obj.GetParent().GetID()
	ingress := obj.(*types.Ingress)
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

func (m *IngressManager) List(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	namespace := obj.GetParent().GetID()
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

func (m *IngressManager) Get(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	namespace := obj.GetParent().GetID()
	ingress := obj.(*types.Ingress)
	k8sIngress, err := getIngress(cluster.KubeClient, namespace, ingress.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get ingress info failed:%s", err.Error())
		}
		return nil
	}

	return k8sIngressToSCIngress(k8sIngress)
}

func (m *IngressManager) Delete(obj resttypes.Object) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := obj.GetParent().GetID()
	ingress := obj.(*types.Ingress)
	err := deleteIngress(cluster.KubeClient, namespace, ingress.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("ingress %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete ingress failed %s", err.Error()))
		}
	}
	return nil
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
	for _, r := range ingress.Rules {
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

	k8sIngress := &extv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ingress.Name,
			Namespace:   namespace,
			Annotations: defaultIngressClassAnnotation,
		},
		Spec: extv1beta1.IngressSpec{
			Rules: rules,
		},
	}
	return cli.Create(context.TODO(), k8sIngress)
}

func deleteIngress(cli client.Client, namespace, name string) error {
	ingress := &extv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), ingress)
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

	ingress := &types.Ingress{
		Name:  k8sIngress.Name,
		Rules: rules,
	}
	ingress.SetID(k8sIngress.Name)
	ingress.SetType(types.IngressType)
	ingress.SetCreationTimestamp(k8sIngress.CreationTimestamp.Time)
	return ingress
}

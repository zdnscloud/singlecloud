package handler

import (
	"context"
	"fmt"
	"sort"

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

	return ingress
}

func (m *IngressManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	err := deleteIngress(cluster.KubeClient, namespace, ctx.Object.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, "ingress doesn't exist")
		}
		return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete ingress failed %s", err.Error()))
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
	if k8sIngress, err := scIngressTok8sIngress(namespace, ingress); err != nil {
		return err
	} else {
		return cli.Create(context.TODO(), k8sIngress)
	}
}

func scIngressTok8sIngress(namespace string, ingress *types.Ingress) (*extv1beta1.Ingress, error) {
	if err := validateAndSortRules(ingress.Rules); err != nil {
		return nil, err
	}

	var httpRules []extv1beta1.IngressRule
	lastHttpRule := -1
	for _, r := range ingress.Rules {
		path := extv1beta1.HTTPIngressPath{
			Path: r.Path,
			Backend: extv1beta1.IngressBackend{
				ServiceName: r.ServiceName,
				ServicePort: intstr.FromInt(r.ServicePort),
			},
		}

		if lastHttpRule != -1 && r.Host == httpRules[lastHttpRule].Host {
			httpRules[lastHttpRule].IngressRuleValue.HTTP.Paths = append(httpRules[lastHttpRule].IngressRuleValue.HTTP.Paths, path)
		} else {
			lastHttpRule += 1
			httpRules = append(httpRules, extv1beta1.IngressRule{
				Host: r.Host,
				IngressRuleValue: extv1beta1.IngressRuleValue{
					HTTP: &extv1beta1.HTTPIngressRuleValue{
						Paths: []extv1beta1.HTTPIngressPath{path},
					},
				},
			})
		}
	}

	k8sIngress := &extv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingress.Name,
			Namespace: namespace,
		},
	}
	k8sIngress.Spec = extv1beta1.IngressSpec{
		Rules: httpRules,
	}
	return k8sIngress, nil
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
		for _, p := range r.IngressRuleValue.HTTP.Paths {
			rules = append(rules, types.IngressRule{
				Host:        r.Host,
				Path:        p.Path,
				ServiceName: p.Backend.ServiceName,
				ServicePort: p.Backend.ServicePort.IntValue(),
			})
		}
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

func validateAndSortRules(rules []types.IngressRule) error {
	sort.Sort(SortedIngressRule(rules))
	ruleCount := len(rules)
	for i := 0; i < ruleCount; i++ {
		r := &rules[i]
		if i+1 < ruleCount {
			if isIngressRuleEqual(r, &rules[i+1]) {
				return fmt.Errorf("has duplicate rule")
			}
		}
		if r.Host == "" {
			return fmt.Errorf("http ingress should have host")
		}

		if r.Path == "" {
			return fmt.Errorf("tcp ingress with empty path")
		}
	}

	return nil
}

type SortedIngressRule []types.IngressRule

func (a SortedIngressRule) Len() int      { return len(a) }
func (a SortedIngressRule) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortedIngressRule) Less(i, j int) bool {
	if a[i].Host != a[j].Host {
		return a[i].Host < a[j].Host
	}
	if a[i].Path != a[j].Path {
		return a[i].Path < a[j].Path
	}
	return false
}

func isIngressRuleEqual(r1, r2 *types.IngressRule) bool {
	return r1.Host == r2.Host && r1.Path == r2.Path
}

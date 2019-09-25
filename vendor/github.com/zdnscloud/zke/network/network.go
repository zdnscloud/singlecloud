package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/network/calico"
	"github.com/zdnscloud/zke/network/coredns"
	"github.com/zdnscloud/zke/network/flannel"
	"github.com/zdnscloud/zke/network/ingress"
	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/pkg/util"

	"github.com/zdnscloud/gok8s/client"
)

const (
	ClusterVersion = "ClusterVersion"
	ClusterCIDR    = "ClusterCIDR"
	CloudProvider  = "CloudProvider"
	RBACConfig     = "RBACConfig"
	KubeCfg        = "KubeCfg"

	NetworkPluginResourceName = "zke-network-plugin"
	NoNetworkPlugin           = "none"

	FlannelNetworkPlugin        = "flannel"
	FlannelIface                = "flannel_iface"
	FlannelBackendType          = "flannel_backend_type"
	FlannelBackendDirectrouting = "flannel_vxlan_directrouting"
	FlannelInterface            = "FlannelInterface"
	FlannelBackend              = "FlannelBackend"

	CalicoNetworkPlugin = "calico"
	CalicoCloudProvider = "calico_cloud_provider"
	Calicoctl           = "Calicoctl"
	CalicoInterface     = "CalicoInterface"

	CoreDNSResourceName = "zke-dns-plugin"

	IngressResourceName = "zke-ingress-plugin"

	Image            = "Image"
	CNIImage         = "CNIImage"
	NodeImage        = "NodeImage"
	ControllersImage = "ControllersImage"

	DeployNamespace = "kube-system"
)

func DeployNetwork(ctx context.Context, c *core.Cluster) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		k8sClient, err := k8s.GetK8sClientFromYaml(c.Certificates[pki.KubeAdminCertName].Config)
		if err != nil {
			return err
		}
		if err := doNetworkPluginDeploy(ctx, c, k8sClient); err != nil {
			return err
		}

		if err := doDNSDeploy(ctx, c, k8sClient); err != nil {
			return err
		}

		if err := doIngressDeploy(ctx, c, k8sClient); err != nil {
			return err
		}
		return nil
	}
}

func doNetworkPluginDeploy(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[network] Setting up network plugin: %s", c.Network.Plugin)
	switch c.Network.Plugin {
	case FlannelNetworkPlugin:
		return doFlannelDeploy(ctx, c, cli)
	case CalicoNetworkPlugin:
		return doCalicoDeploy(ctx, c, cli)
	case NoNetworkPlugin:
		log.Infof(ctx, "[Network] Not deploying a cluster network, expecting custom CNI")
		return nil
	default:
		return fmt.Errorf("[Network] Unsupported network plugin: %s", c.Network.Plugin)
	}
}

func doFlannelDeploy(ctx context.Context, c *core.Cluster, cli client.Client) error {
	flannelConfig := map[string]interface{}{
		ClusterCIDR:      c.Option.ClusterCidr,
		Image:            c.Image.Flannel,
		CNIImage:         c.Image.FlannelCNI,
		FlannelInterface: c.Network.Iface,
		FlannelBackend: map[string]interface{}{
			"Type": "vxlan",
		},
		RBACConfig:        c.Authorization.Mode,
		ClusterVersion:    core.GetTagMajorVersion(c.Option.KubernetesVersion),
		"DeployNamespace": DeployNamespace,
	}
	if err := k8s.DoCreateFromTemplate(cli, flannel.FlannelTemplate, flannelConfig); err != nil {
		return err
	}
	log.Infof(ctx, "[Network] network plugin flannel deployed successfully")
	return nil
}

func doCalicoDeploy(ctx context.Context, c *core.Cluster, cli client.Client) error {
	clientConfig := pki.GetConfigPath(pki.KubeNodeCertName)
	calicoConfig := map[string]interface{}{
		KubeCfg:           clientConfig,
		ClusterCIDR:       c.Option.ClusterCidr,
		CalicoInterface:   c.Network.Iface,
		CNIImage:          c.Image.CalicoCNI,
		NodeImage:         c.Image.CalicoNode,
		Calicoctl:         c.Image.CalicoCtl,
		CloudProvider:     "none",
		RBACConfig:        c.Authorization.Mode,
		"DeployNamespace": DeployNamespace,
	}

	if err := k8s.DoCreateFromTemplate(cli, calico.CalicoTemplateV113, calicoConfig); err != nil {
		return err
	}
	log.Infof(ctx, "[Network] network plugin calico deployed successfully")
	return nil
}

func doDNSDeploy(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[DNS] Setting up DNS plugin %s", c.Network.DNS.Provider)
	CoreDNSConfig := coredns.CoreDNSOptions{
		CoreDNSImage:           c.Image.CoreDNS,
		CoreDNSAutoScalerImage: c.Image.CoreDNSAutoscaler,
		RBACConfig:             c.Authorization.Mode,
		ClusterDomain:          c.Option.ClusterDomain,
		ClusterDNSServer:       c.Option.ClusterDNSServiceIP,
		UpstreamNameservers:    c.Network.DNS.UpstreamNameservers,
		ReverseCIDRs:           c.Network.DNS.ReverseCIDRs,
	}
	if err := k8s.DoCreateFromTemplate(cli, coredns.CoreDNSTemplate, CoreDNSConfig); err != nil {
		return err
	}
	log.Infof(ctx, "[Network] DNS plugin coredns deployed successfully")
	return nil
}

func doIngressDeploy(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[Network] Setting up %s ingress controller", c.Network.Ingress.Provider)
	ingressConfig := ingress.IngressOptions{
		RBACConfig:     c.Authorization.Mode,
		Options:        c.Network.Ingress.Options,
		NodeSelector:   c.Network.Ingress.NodeSelector,
		ExtraArgs:      c.Network.Ingress.ExtraArgs,
		IngressImage:   c.Image.Ingress,
		IngressBackend: c.Image.IngressBackend,
	}
	ingressSplits := strings.SplitN(c.Image.Ingress, ":", 2)
	if len(ingressSplits) == 2 {
		version := strings.Split(ingressSplits[1], "-")[0]
		if version < "0.16.0" {
			ingressConfig.AlpineImage = c.Image.Alpine
		}
	}
	if err := k8s.DoCreateFromTemplate(cli, ingress.NginxIngressTemplate, ingressConfig); err != nil {
		return err
	}
	log.Infof(ctx, "[Network] ingress controller %s deployed successfully", c.Network.Ingress.Provider)
	return nil
}

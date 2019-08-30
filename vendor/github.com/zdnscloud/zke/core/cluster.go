package core

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/zdnscloud/zke/core/authz"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/core/services"
	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/pkg/util"
	"github.com/zdnscloud/zke/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/gok8s/client/config"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
)

type Cluster struct {
	types.ZKEConfig      `yaml:",inline"`
	KubeConfig           string
	AuthnStrategies      map[string]bool
	Certificates         map[string]pki.CertificatePKI
	DockerDialerFactory  hosts.DialerFactory
	K8sWrapTransport     k8s.WrapTransport
	KubeClient           *kubernetes.Clientset
	KubernetesServiceIP  net.IP
	PrivateRegistriesMap map[string]types.PrivateRegistry
	ControlPlaneHosts    []*hosts.Host
	EtcdHosts            []*hosts.Host
	EtcdReadyHosts       []*hosts.Host
	InactiveHosts        []*hosts.Host
	WorkerHosts          []*hosts.Host
	EdgeHosts            []*hosts.Host
}

const (
	AuthnX509Provider       = "x509"
	AuthnWebhookProvider    = "webhook"
	StateConfigMapName      = "cluster-state"
	ClusterConfigMapName    = "cluster-config"
	UpdateStateTimeout      = 30
	GetStateTimeout         = 30
	KubernetesClientTimeOut = 30
	NoneAuthorizationMode   = "none"
	LocalNodeAddress        = "127.0.0.1"
	LocalNodeHostname       = "localhost"
	LocalNodeUser           = "root"
	ControlPlane            = "controlPlane"
	WorkerPlane             = "workerPlan"
	EtcdPlane               = "etcd"

	KubeAppLabel = "k8s-app"
	AppLabel     = "app"
	NameLabel    = "name"
)

func (c *Cluster) DeployControlPlane(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		// Deploy Etcd Plane
		etcdNodePlanMap := make(map[string]types.ZKENodePlan)
		// Build etcd node plan map
		for _, etcdHost := range c.EtcdHosts {
			etcdNodePlanMap[etcdHost.Address] = BuildZKEConfigNodePlan(ctx, c, etcdHost, etcdHost.DockerInfo)
		}
		if len(c.Core.Etcd.ExternalURLs) > 0 {
			log.Infof(ctx, "[etcd] External etcd connection string has been specified, skipping etcd plane")
		} else {
			if err := services.RunEtcdPlane(ctx, c.EtcdHosts, etcdNodePlanMap, c.PrivateRegistriesMap, c.Image.Alpine, c.Core.Etcd, c.Certificates); err != nil {
				return fmt.Errorf("[etcd] Failed to bring up Etcd Plane: %v", err)
			}
		}
		// Deploy Control plane
		cpNodePlanMap := make(map[string]types.ZKENodePlan)
		// Build cp node plan map
		for _, cpHost := range c.ControlPlaneHosts {
			cpNodePlanMap[cpHost.Address] = BuildZKEConfigNodePlan(ctx, c, cpHost, cpHost.DockerInfo)
		}
		if err := services.RunControlPlane(ctx, c.ControlPlaneHosts,
			c.PrivateRegistriesMap,
			cpNodePlanMap,
			c.Image.Alpine,
			c.Certificates); err != nil {
			return fmt.Errorf("[controlPlane] Failed to bring up Control Plane: %v", err)
		}
		return nil
	}
}

func (c *Cluster) DeployWorkerPlane(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		// Deploy Worker plane
		workerNodePlanMap := make(map[string]types.ZKENodePlan)
		// Build cp node plan map
		allHosts := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)
		for _, workerHost := range allHosts {
			workerNodePlanMap[workerHost.Address] = BuildZKEConfigNodePlan(ctx, c, workerHost, workerHost.DockerInfo)
		}
		if err := services.RunWorkerPlane(ctx, allHosts,
			c.PrivateRegistriesMap,
			workerNodePlanMap,
			c.Certificates,
			c.Image.Alpine); err != nil {
			return fmt.Errorf("[workerPlane] Failed to bring up Worker Plane: %v", err)
		}
		return nil
	}
}

func ParseConfig(clusterFile string) (*types.ZKEConfig, error) {
	log.Debugf("Parsing cluster file [%v]", clusterFile)
	var zkeConfig types.ZKEConfig
	if err := yaml.Unmarshal([]byte(clusterFile), &zkeConfig); err != nil {
		return nil, err
	}
	return &zkeConfig, nil
}

func InitClusterObject(ctx context.Context, zkeConfig *types.ZKEConfig) (*Cluster, error) {
	select {
	case <-ctx.Done():
		return nil, util.CancelErr
	default:
		// basic cluster object from zkeConfig
		c := &Cluster{
			AuthnStrategies:      make(map[string]bool),
			ZKEConfig:            *zkeConfig,
			PrivateRegistriesMap: make(map[string]types.PrivateRegistry),
		}

		// Setting cluster Defaults
		err := c.setClusterDefaults(ctx)
		if err != nil {
			return nil, err
		}
		// set cluster kubernetesService ip
		c.KubernetesServiceIP, err = pki.GetKubernetesServiceIP(c.Core.KubeAPI.ServiceClusterIPRange)
		if err != nil {
			return c, fmt.Errorf("Failed to get Kubernetes Service IP: %v", err)
		}

		// set hosts groups
		if err := c.InvertIndexHosts(); err != nil {
			return nil, fmt.Errorf("Failed to classify hosts from config file: %v", err)
		}
		// validate cluster configuration
		if err := c.ValidateCluster(); err != nil {
			return nil, fmt.Errorf("Failed to validate cluster: %v", err)
		}
		return c, nil
	}
}

func (c *Cluster) SetupDialers(ctx context.Context, dailersOptions hosts.DialersOptions) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		c.DockerDialerFactory = dailersOptions.DockerDialerFactory
		c.K8sWrapTransport = dailersOptions.K8sWrapTransport
		return nil
	}
}

func RebuildKubeconfig(ctx context.Context, kubeCluster *Cluster, clusterState *FullState) error {
	return rebuildLocalAdminConfig(ctx, kubeCluster)
}

func rebuildLocalAdminConfig(ctx context.Context, kubeCluster *Cluster) error {
	if len(kubeCluster.ControlPlaneHosts) == 0 {
		return nil
	}
	log.Infof(ctx, "[reconcile] Rebuilding and updating local kube config")
	var workingConfig, newConfig string
	currentKubeConfig := kubeCluster.Certificates[pki.KubeAdminCertName]
	caCrt := kubeCluster.Certificates[pki.CACertName].Certificate
	for _, cpHost := range kubeCluster.ControlPlaneHosts {
		kubeURL := fmt.Sprintf("https://%s:6443", cpHost.Address)
		caData := string(cert.EncodeCertPEM(caCrt))
		crtData := string(cert.EncodeCertPEM(currentKubeConfig.Certificate))
		keyData := string(cert.EncodePrivateKeyPEM(currentKubeConfig.Key))
		newConfig = pki.GetKubeConfigX509WithData(kubeURL, kubeCluster.ClusterName, pki.KubeAdminCertName, caData, crtData, keyData)

		workingConfig = newConfig
		kubeConfig, err := config.BuildConfig([]byte(newConfig))
		if err != nil {
			return err
		}
		kubeClientSet, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			return err
		}
		kubeCluster.KubeClient = kubeClientSet
		if _, err := GetK8sVersion(kubeClientSet); err == nil {
			log.Infof(ctx, "[reconcile] host [%s] is active master on the cluster", cpHost.Address)
			break
		}
	}
	currentKubeConfig.Config = workingConfig
	kubeCluster.Certificates[pki.KubeAdminCertName] = currentKubeConfig

	return nil
}

func getLocalConfigAddress(kubeConfigYaml string) (string, error) {
	config, err := config.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		return "", err
	}
	splittedAdress := strings.Split(config.Host, ":")
	address := splittedAdress[1]
	return address[2:], nil
}

func getLocalAdminConfigWithNewAddress(kubeConfigYaml, cpAddress string, clusterName string) string {
	config, err := config.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		return ""
	}
	config.Host = fmt.Sprintf("https://%s:6443", cpAddress)
	return pki.GetKubeConfigX509WithData(
		"https://"+cpAddress+":6443",
		clusterName,
		pki.KubeAdminCertName,
		string(config.CAData),
		string(config.CertData),
		string(config.KeyData))
}

func ApplyAuthzResources(ctx context.Context, zkeConfig types.ZKEConfig, k8sClient *kubernetes.Clientset, dailersOptions hosts.DialersOptions) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		// dialer factories are not needed here since we are not uses docker only k8s jobs
		kubeCluster, err := InitClusterObject(ctx, &zkeConfig)
		if err != nil {
			return err
		}
		if err := kubeCluster.SetupDialers(ctx, dailersOptions); err != nil {
			return err
		}
		if len(kubeCluster.ControlPlaneHosts) == 0 {
			return nil
		}
		if err := authz.ApplyJobDeployerServiceAccount(ctx, k8sClient); err != nil {
			return fmt.Errorf("Failed to apply the ServiceAccount needed for job execution: %v", err)
		}
		if kubeCluster.Authorization.Mode == NoneAuthorizationMode {
			return nil
		}
		if kubeCluster.Authorization.Mode == services.RBACAuthorizationMode {
			if err := authz.ApplySystemNodeClusterRoleBinding(ctx, k8sClient); err != nil {
				return fmt.Errorf("Failed to apply the ClusterRoleBinding needed for node authorization: %v", err)
			}
		}
		if kubeCluster.Authorization.Mode == services.RBACAuthorizationMode && kubeCluster.Core.KubeAPI.PodSecurityPolicy {
			if err := authz.ApplyDefaultPodSecurityPolicy(ctx, k8sClient); err != nil {
				return fmt.Errorf("Failed to apply default PodSecurityPolicy: %v", err)
			}
			if err := authz.ApplyDefaultPodSecurityPolicyRole(ctx, k8sClient); err != nil {
				return fmt.Errorf("Failed to apply default PodSecurityPolicy ClusterRole and ClusterRoleBinding: %v", err)
			}
		}
		return nil
	}
}

func (c *Cluster) SyncLabelsAndTaints(ctx context.Context, currentCluster *Cluster) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		if currentCluster != nil {
			cpToDelete := hosts.GetToDeleteHosts(currentCluster.ControlPlaneHosts, c.ControlPlaneHosts, c.InactiveHosts)
			if len(cpToDelete) == len(currentCluster.ControlPlaneHosts) {
				log.Infof(ctx, "[sync] Cleaning left control plane nodes from reconcilation")
				for _, toDeleteHost := range cpToDelete {
					if err := cleanControlNode(ctx, c, currentCluster, toDeleteHost); err != nil {
						return err
					}
				}
			}
		}
	}

	if len(c.ControlPlaneHosts) > 0 {
		log.Infof(ctx, "[sync] Syncing nodes Labels and Taints")
		hostList := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)
		_, err := errgroup.Batch(hostList, func(h interface{}) (interface{}, error) {
			log.Debugf("worker starting sync for node [%s]", h.(*hosts.Host).NodeName)
			return nil, setNodeAnnotationsLabelsTaints(c.KubeClient, h.(*hosts.Host))
		})

		if err != nil {
			return err
		}
		log.Infof(ctx, "[sync] Successfully synced nodes Labels and Taints")
	}
	return nil
}

func setNodeAnnotationsLabelsTaints(k8sClient *kubernetes.Clientset, host *hosts.Host) error {
	node := &v1.Node{}
	var err error
	for retries := 0; retries <= 5; retries++ {
		node, err = k8s.GetNode(k8sClient, host.NodeName)
		if err != nil {
			log.Debugf("[hosts] Can't find node by name [%s], retrying..", host.NodeName)
			time.Sleep(2 * time.Second)
			continue
		}

		oldNode := node.DeepCopy()
		k8s.SetNodeAddressesAnnotations(node, host.InternalAddress, host.Address)
		k8s.SyncNodeLabels(node, host.ToAddLabels, host.ToDelLabels)
		k8s.SyncNodeTaints(node, host.ToAddTaints, host.ToDelTaints)

		if reflect.DeepEqual(oldNode, node) {
			log.Debugf("skipping syncing labels for node [%s]", node.Name)
			return nil
		}
		_, err = k8sClient.CoreV1().Nodes().Update(node)
		if err != nil {
			log.Debugf("Error syncing labels for node [%s]: %v", node.Name, err)
			time.Sleep(5 * time.Second)
			continue
		}
		return nil
	}
	return err
}

func (c *Cluster) PrePullK8sImages(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		log.Infof(ctx, "Pre-pulling kubernetes images")
		hostList := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)
		_, err := errgroup.Batch(hostList, func(h interface{}) (interface{}, error) {
			runHost := h.(*hosts.Host)
			return nil, docker.UseLocalOrPull(ctx, runHost.DClient, runHost.Address, c.Image.Kubernetes, "pre-deploy", c.PrivateRegistriesMap)
		})
		if err != nil {
			return err
		}
		log.Infof(ctx, "Kubernetes images pulled successfully")
		return nil
	}
}

func RestartClusterPods(ctx context.Context, kubeCluster *Cluster) error {
	log.Infof(ctx, "Restarting network, ingress, and metrics pods")
	// this will remove the pods created by ZKE and let the controller creates them again

	labelsList := []string{
		fmt.Sprintf("%s=%s", KubeAppLabel, CalicoNetworkPlugin),
		fmt.Sprintf("%s=%s", KubeAppLabel, FlannelNetworkPlugin),
		fmt.Sprintf("%s=%s", AppLabel, NginxIngressAddonAppName),
		fmt.Sprintf("%s=%s", KubeAppLabel, DefaultMonitorMetricsProvider),
		fmt.Sprintf("%s=%s", KubeAppLabel, CoreDNSAddonAppName),
	}

	_, err := errgroup.Batch(labelsList, func(l interface{}) (interface{}, error) {
		runLabel := l.(string)
		// list pods to be deleted
		pods, err := k8s.ListPodsByLabel(kubeCluster.KubeClient, runLabel)
		if err != nil {
			return nil, err
		}
		// delete pods
		err = k8s.DeletePods(kubeCluster.KubeClient, pods)
		return nil, err
	})
	return err
}

func (c *Cluster) GetHostInfoMap() map[string]dockertypes.Info {
	hostsInfoMap := make(map[string]dockertypes.Info)
	allHosts := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)
	for _, host := range allHosts {
		hostsInfoMap[host.Address] = host.DockerInfo
	}
	return hostsInfoMap
}

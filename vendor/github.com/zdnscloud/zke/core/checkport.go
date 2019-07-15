package core

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"github.com/zdnscloud/cement/errgroup"
)

const (
	PortCheckContainer        = "zke-port-checker"
	EtcdPortListenContainer   = "zke-etcd-port-listener"
	CPPortListenContainer     = "zke-cp-port-listener"
	WorkerPortListenContainer = "zke-worker-port-listener"

	KubeAPIPort         = "6443"
	EtcdPort1           = "2379"
	EtcdPort2           = "2380"
	ScedulerPort        = "10251"
	ControllerPort      = "10252"
	KubeletPort         = "10250"
	KubeProxyPort       = "10256"
	FlannetVXLANPortUDP = "8472"

	ProtocolTCP = "TCP"
	ProtocolUDP = "UDP"

	NoNetworkPlugin = "none"

	FlannelNetworkPlugin = "flannel"
	CalicoNetworkPlugin  = "calico"
	CalicoCloudProvider  = "calico_cloud_provider"
	FlannelInterface     = "FlannelInterface"
	FlannelBackend       = "FlannelBackend"

	FlannelIface                = "flannel_iface"
	FlannelBackendType          = "flannel_backend_type"
	FlannelBackendDirectrouting = "flannel_vxlan_directrouting"

	// EtcdEndpoints is the server address for Etcd, used by calico
	EtcdEndpoints = "EtcdEndpoints"
	// APIRoot is the kubernetes API address
	APIRoot = "APIRoot"
	// kubernetes client certificates and kubeconfig paths
	EtcdClientCert     = "EtcdClientCert"
	EtcdClientKey      = "EtcdClientKey"
	EtcdClientCA       = "EtcdClientCA"
	EtcdClientCertPath = "EtcdClientCertPath"
	EtcdClientKeyPath  = "EtcdClientKeyPath"
	EtcdClientCAPath   = "EtcdClientCAPath"

	ClientCertPath = "ClientCertPath"
	ClientKeyPath  = "ClientKeyPath"
	ClientCAPath   = "ClientCAPath"

	RBACConfig     = "RBACConfig"
	ClusterVersion = "ClusterVersion"
)

var EtcdPortList = []string{
	EtcdPort1,
	EtcdPort2,
}

var ControlPlanePortList = []string{
	KubeAPIPort,
}

var WorkerPortList = []string{
	KubeletPort,
}

var EtcdClientPortList = []string{
	EtcdPort1,
}

func (c *Cluster) CheckClusterPorts(ctx context.Context, currentCluster *Cluster) error {
	if currentCluster != nil {
		newEtcdHost := hosts.GetToAddHosts(currentCluster.EtcdHosts, c.EtcdHosts)
		newControlPlanHosts := hosts.GetToAddHosts(currentCluster.ControlPlaneHosts, c.ControlPlaneHosts)
		newWorkerHosts := hosts.GetToAddHosts(currentCluster.WorkerHosts, c.WorkerHosts)

		if len(newEtcdHost) == 0 &&
			len(newWorkerHosts) == 0 &&
			len(newControlPlanHosts) == 0 {
			log.Infof(ctx, "[network] No hosts added existing cluster, skipping port check")
			return nil
		}
	}
	if err := c.deployTCPPortListeners(ctx, currentCluster); err != nil {
		return err
	}
	if err := c.runServicePortChecks(ctx); err != nil {
		return err
	}
	// Skip kubeapi check if we are using custom k8s dialer
	if c.K8sWrapTransport == nil {
		if err := c.checkKubeAPIPort(ctx); err != nil {
			return err
		}
	} else {
		log.Infof(ctx, "[network] Skipping kubeapi port check")
	}

	return c.removeTCPPortListeners(ctx)
}

func (c *Cluster) checkKubeAPIPort(ctx context.Context) error {
	log.Infof(ctx, "[network] Checking KubeAPI port Control Plane hosts")
	for _, host := range c.ControlPlaneHosts {
		log.Debugf("[network] Checking KubeAPI port [%s] on host: %s", KubeAPIPort, host.Address)
		address := fmt.Sprintf("%s:%s", host.Address, KubeAPIPort)
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return fmt.Errorf("[network] Can't access KubeAPI port [%s] on Control Plane host: %s", KubeAPIPort, host.Address)
		}
		conn.Close()
	}
	return nil
}

func (c *Cluster) deployTCPPortListeners(ctx context.Context, currentCluster *Cluster) error {
	log.Infof(ctx, "[network] Deploying port listener containers")

	// deploy ectd listeners
	if err := c.deployListenerOnPlane(ctx, EtcdPortList, c.EtcdHosts, EtcdPortListenContainer); err != nil {
		return err
	}

	// deploy controlplane listeners
	if err := c.deployListenerOnPlane(ctx, ControlPlanePortList, c.ControlPlaneHosts, CPPortListenContainer); err != nil {
		return err
	}

	// deploy worker listeners
	if err := c.deployListenerOnPlane(ctx, WorkerPortList, c.WorkerHosts, WorkerPortListenContainer); err != nil {
		return err
	}
	log.Infof(ctx, "[network] Port listener containers deployed successfully")
	return nil
}

func (c *Cluster) deployListenerOnPlane(ctx context.Context, portList []string, hostPlane []*hosts.Host, containerName string) error {

	_, err := errgroup.Batch(hostPlane, func(h interface{}) (interface{}, error) {
		return nil, c.deployListener(ctx, h.(*hosts.Host), portList, containerName)
	})
	return err
}

func (c *Cluster) deployListener(ctx context.Context, host *hosts.Host, portList []string, containerName string) error {
	imageCfg := &container.Config{
		Image: c.Image.Alpine,
		Cmd: []string{
			"nc",
			"-kl",
			"-p",
			"1337",
			"-e",
			"echo",
		},
		ExposedPorts: nat.PortSet{
			"1337/tcp": {},
		},
	}
	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			"1337/tcp": getPortBindings("0.0.0.0", portList),
		},
	}

	log.Debugf("[network] Starting deployListener [%s] on host [%s]", containerName, host.Address)
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, containerName, host.Address, "network", c.PrivateRegistriesMap); err != nil {
		if strings.Contains(err.Error(), "bind: address already in use") {
			log.Debugf("[network] Service is already up on host [%s]", host.Address)
			return nil
		}
		return err
	}
	return nil
}

func (c *Cluster) removeTCPPortListeners(ctx context.Context) error {
	log.Infof(ctx, "[network] Removing port listener containers")

	if err := removeListenerFromPlane(ctx, c.EtcdHosts, EtcdPortListenContainer); err != nil {
		return err
	}
	if err := removeListenerFromPlane(ctx, c.ControlPlaneHosts, CPPortListenContainer); err != nil {
		return err
	}
	if err := removeListenerFromPlane(ctx, c.WorkerHosts, WorkerPortListenContainer); err != nil {
		return err
	}
	log.Infof(ctx, "[network] Port listener containers removed successfully")
	return nil
}

func removeListenerFromPlane(ctx context.Context, hostPlane []*hosts.Host, containerName string) error {
	_, err := errgroup.Batch(hostPlane, func(h interface{}) (interface{}, error) {
		runHost := h.(*hosts.Host)
		return nil, docker.DoRemoveContainer(ctx, runHost.DClient, containerName, runHost.Address)
	})
	return err
}

func (c *Cluster) runServicePortChecks(ctx context.Context) error {
	// check etcd <-> etcd
	// one etcd host is a pass
	if len(c.EtcdHosts) > 1 {
		log.Infof(ctx, "[network] Running etcd <-> etcd port checks")
		_, err := errgroup.Batch(c.EtcdHosts, func(h interface{}) (interface{}, error) {
			return nil, checkPlaneTCPPortsFromHost(ctx, h.(*hosts.Host), EtcdPortList, c.EtcdHosts, c.Image.Alpine, c.PrivateRegistriesMap)
		})
		return err
	}
	// check control -> etcd connectivity
	log.Infof(ctx, "[network] Running control plane -> etcd port checks")
	_, err := errgroup.Batch(c.ControlPlaneHosts, func(h interface{}) (interface{}, error) {
		return nil, checkPlaneTCPPortsFromHost(ctx, h.(*hosts.Host), EtcdClientPortList, c.EtcdHosts, c.Image.Alpine, c.PrivateRegistriesMap)
	})
	if err != nil {
		return err
	}
	// check controle plane -> Workers
	log.Infof(ctx, "[network] Running control plane -> worker port checks")
	_, err = errgroup.Batch(c.ControlPlaneHosts, func(h interface{}) (interface{}, error) {
		return nil, checkPlaneTCPPortsFromHost(ctx, h.(*hosts.Host), WorkerPortList, c.WorkerHosts, c.Image.Alpine, c.PrivateRegistriesMap)
	})
	if err != nil {
		return err
	}
	// check workers -> control plane
	log.Infof(ctx, "[network] Running workers -> control plane port checks")
	_, err = errgroup.Batch(c.WorkerHosts, func(h interface{}) (interface{}, error) {
		return nil, checkPlaneTCPPortsFromHost(ctx, h.(*hosts.Host), ControlPlanePortList, c.ControlPlaneHosts, c.Image.Alpine, c.PrivateRegistriesMap)
	})
	return err
}

func checkPlaneTCPPortsFromHost(ctx context.Context, host *hosts.Host, portList []string, planeHosts []*hosts.Host, image string, prsMap map[string]types.PrivateRegistry) error {
	var hosts []string

	for _, host := range planeHosts {
		hosts = append(hosts, host.InternalAddress)
	}
	imageCfg := &container.Config{
		Image: image,
		Env: []string{
			fmt.Sprintf("HOSTS=%s", strings.Join(hosts, " ")),
			fmt.Sprintf("PORTS=%s", strings.Join(portList, " ")),
		},
		Cmd: []string{
			"sh",
			"-c",
			"for host in $HOSTS; do for port in $PORTS ; do echo \"Checking host ${host} on port ${port}\" >&1 & nc -w 5 -z $host $port > /dev/null || echo \"${host}:${port}\" >&2 & done; wait; done",
		},
	}
	hostCfg := &container.HostConfig{
		NetworkMode: "host",
		LogConfig: container.LogConfig{
			Type: "json-file",
		},
	}
	if err := docker.DoRemoveContainer(ctx, host.DClient, PortCheckContainer, host.Address); err != nil {
		return err
	}
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, PortCheckContainer, host.Address, "network", prsMap); err != nil {
		return err
	}

	containerLog, _, logsErr := docker.GetContainerLogsStdoutStderr(ctx, host.DClient, PortCheckContainer, "all", true)
	if logsErr != nil {
		log.Warnf(ctx, "[network] Failed to get network port check logs: %v", logsErr)
	}
	log.Debugf("[network] containerLog [%s] on host: %s", containerLog, host.Address)

	if err := docker.RemoveContainer(ctx, host.DClient, host.Address, PortCheckContainer); err != nil {
		return err
	}
	log.Debugf("[network] Length of containerLog is [%d] on host: %s", len(containerLog), host.Address)
	if len(containerLog) > 0 {
		portCheckLogs := strings.Join(strings.Split(strings.TrimSpace(containerLog), "\n"), ", ")
		return fmt.Errorf("[network] Host [%s] is not able to connect to the following ports: [%s]. Please check network policies and firewall rules", host.Address, portCheckLogs)
	}
	return nil
}

func getPortBindings(hostAddress string, portList []string) []nat.PortBinding {
	portBindingList := []nat.PortBinding{}
	for _, portNumber := range portList {
		rawPort := fmt.Sprintf("%s:%s:1337/tcp", hostAddress, portNumber)
		portMapping, _ := nat.ParsePortSpec(rawPort)
		portBindingList = append(portBindingList, portMapping[0].Binding)
	}
	return portBindingList
}

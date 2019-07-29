package core

import (
	"context"
	"fmt"

	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/core/services"
	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
)

const (
	unschedulableEtcdTaint    = "node-role.kubernetes.io/etcd=true:NoExecute"
	unschedulableControlTaint = "node-role.kubernetes.io/controlplane=true:NoSchedule"
)

func ReconcileCluster(ctx context.Context, kubeCluster, currentCluster *Cluster) error {
	log.Infof(ctx, "[reconcile] Reconciling cluster state")
	if currentCluster == nil {
		log.Infof(ctx, "[reconcile] This is newly generated cluster")
		return nil
	}
	// sync node labels to define the toDelete labels
	syncLabels(ctx, currentCluster, kubeCluster)

	if err := reconcileEtcd(ctx, currentCluster, kubeCluster, kubeCluster.KubeClient); err != nil {
		return fmt.Errorf("Failed to reconcile etcd plane: %v", err)
	}

	if err := reconcileWorker(ctx, currentCluster, kubeCluster, kubeCluster.KubeClient); err != nil {
		return err
	}

	if err := reconcileControl(ctx, currentCluster, kubeCluster, kubeCluster.KubeClient); err != nil {
		return err
	}

	log.Infof(ctx, "[reconcile] Reconciled cluster state successfully")
	return nil
}

func reconcileWorker(ctx context.Context, currentCluster, kubeCluster *Cluster, kubeClient *kubernetes.Clientset) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("cluster build has beed canceled")
	default:
		// worker deleted first to avoid issues when worker+controller on same host
		log.Debugf("[reconcile] Check worker hosts to be deleted")
		wpToDelete := hosts.GetToDeleteHosts(currentCluster.WorkerHosts, kubeCluster.WorkerHosts, kubeCluster.InactiveHosts)
		for _, toDeleteHost := range wpToDelete {
			toDeleteHost.IsWorker = false
			if err := hosts.DeleteNode(ctx, toDeleteHost, kubeClient, toDeleteHost.IsControl); err != nil {
				return fmt.Errorf("Failed to delete worker node [%s] from cluster: %v", toDeleteHost.Address, err)
			}
			// attempting to clean services/files on the host
			if err := reconcileHost(ctx, toDeleteHost, true, false, currentCluster.Image.Alpine, currentCluster.DockerDialerFactory, currentCluster.PrivateRegistriesMap, currentCluster.Option.PrefixPath, currentCluster.Option.KubernetesVersion); err != nil {
				log.Warnf(ctx, "[reconcile] Couldn't clean up worker node [%s]: %v", toDeleteHost.Address, err)
				continue
			}
		}
		// attempt to remove unschedulable taint
		toAddHosts := hosts.GetToAddHosts(currentCluster.WorkerHosts, kubeCluster.WorkerHosts)
		for _, host := range toAddHosts {
			host.UpdateWorker = true
			if host.IsEtcd {
				host.ToDelTaints = append(host.ToDelTaints, unschedulableEtcdTaint)
			}
			if host.IsControl {
				host.ToDelTaints = append(host.ToDelTaints, unschedulableControlTaint)
			}
		}
		return nil
	}
}

func reconcileControl(ctx context.Context, currentCluster, kubeCluster *Cluster, kubeClient *kubernetes.Clientset) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("cluster build has beed canceled")
	default:
		log.Debugf("[reconcile] Check Control plane hosts to be deleted")
		selfDeleteAddress, err := getLocalConfigAddress(kubeCluster.Certificates[pki.KubeAdminCertName].Config)
		if err != nil {
			return err
		}
		cpToDelete := hosts.GetToDeleteHosts(currentCluster.ControlPlaneHosts, kubeCluster.ControlPlaneHosts, kubeCluster.InactiveHosts)
		// move the current host in local kubeconfig to the end of the list
		for i, toDeleteHost := range cpToDelete {
			if toDeleteHost.Address == selfDeleteAddress {
				cpToDelete = append(cpToDelete[:i], cpToDelete[i+1:]...)
				cpToDelete = append(cpToDelete, toDeleteHost)
			}
		}
		if len(cpToDelete) == len(currentCluster.ControlPlaneHosts) {
			log.Infof(ctx, "[reconcile] Deleting all current controlplane nodes, skipping deleting from k8s cluster")
			// rebuilding local admin config to enable saving cluster state
			// if err := RebuildKubeconfigForRest(ctx, kubeCluster); err != nil {
			if err := rebuildLocalAdminConfig(ctx, kubeCluster); err != nil {
				return err
			}
			return nil
		}
		for _, toDeleteHost := range cpToDelete {
			if err := cleanControlNode(ctx, kubeCluster, currentCluster, toDeleteHost); err != nil {
				return err
			}
		}
		// rebuilding local admin config to enable saving cluster state
		// if err := RebuildKubeconfigForRest(ctx, kubeCluster); err != nil {
		if err := rebuildLocalAdminConfig(ctx, kubeCluster); err != nil {
			return err
		}
		return nil
	}
}

func reconcileHost(ctx context.Context, toDeleteHost *hosts.Host, worker, etcd bool, cleanerImage string, dialerFactory hosts.DialerFactory, prsMap map[string]types.PrivateRegistry, clusterPrefixPath string, clusterVersion string) error {
	if err := toDeleteHost.TunnelUp(ctx, dialerFactory, clusterPrefixPath, clusterVersion); err != nil {
		return fmt.Errorf("Not able to reach the host: %v", err)
	}
	if worker {
		if err := services.RemoveWorkerPlane(ctx, []*hosts.Host{toDeleteHost}, false); err != nil {
			return fmt.Errorf("Couldn't remove worker plane: %v", err)
		}
		if err := toDeleteHost.CleanUpWorkerHost(ctx, cleanerImage, prsMap); err != nil {
			return fmt.Errorf("Not able to clean the host: %v", err)
		}
	} else if etcd {
		if err := services.RemoveEtcdPlane(ctx, []*hosts.Host{toDeleteHost}, false); err != nil {
			return fmt.Errorf("Couldn't remove etcd plane: %v", err)
		}
		if err := toDeleteHost.CleanUpEtcdHost(ctx, cleanerImage, prsMap); err != nil {
			return fmt.Errorf("Not able to clean the host: %v", err)
		}
	} else {
		if err := services.RemoveControlPlane(ctx, []*hosts.Host{toDeleteHost}, false); err != nil {
			return fmt.Errorf("Couldn't remove control plane: %v", err)
		}
		if err := toDeleteHost.CleanUpControlHost(ctx, cleanerImage, prsMap); err != nil {
			return fmt.Errorf("Not able to clean the host: %v", err)
		}
	}
	return nil
}

func reconcileEtcd(ctx context.Context, currentCluster, kubeCluster *Cluster, kubeClient *kubernetes.Clientset) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("cluster build has beed canceled")
	default:
		log.Infof(ctx, "[reconcile] Check etcd hosts to be deleted")
		// get tls for the first current etcd host
		clientCert := cert.EncodeCertPEM(currentCluster.Certificates[pki.KubeNodeCertName].Certificate)
		clientkey := cert.EncodePrivateKeyPEM(currentCluster.Certificates[pki.KubeNodeCertName].Key)

		etcdToDelete := hosts.GetToDeleteHosts(currentCluster.EtcdHosts, kubeCluster.EtcdHosts, kubeCluster.InactiveHosts)
		for _, etcdHost := range etcdToDelete {
			if err := services.RemoveEtcdMember(ctx, etcdHost, kubeCluster.EtcdHosts, clientCert, clientkey); err != nil {
				log.Warnf(ctx, "[reconcile] %v", err)
				continue
			}
			if err := hosts.DeleteNode(ctx, etcdHost, kubeClient, etcdHost.IsControl); err != nil {
				log.Warnf(ctx, "Failed to delete etcd node [%s] from cluster: %v", etcdHost.Address, err)
				continue
			}
			// attempting to clean services/files on the host
			if err := reconcileHost(ctx, etcdHost, false, true, currentCluster.Image.Alpine, currentCluster.DockerDialerFactory, currentCluster.PrivateRegistriesMap, currentCluster.Option.PrefixPath, currentCluster.Option.KubernetesVersion); err != nil {
				log.Warnf(ctx, "[reconcile] Couldn't clean up etcd node [%s]: %v", etcdHost.Address, err)
				continue
			}
		}
		log.Infof(ctx, "[reconcile] Check etcd hosts to be added")
		etcdToAdd := hosts.GetToAddHosts(currentCluster.EtcdHosts, kubeCluster.EtcdHosts)
		for _, etcdHost := range etcdToAdd {
			etcdHost.ToAddEtcdMember = true
		}
		for _, etcdHost := range etcdToAdd {
			// Check if the host already part of the cluster -- this will cover cluster with lost quorum
			isEtcdMember, err := services.IsEtcdMember(ctx, etcdHost, kubeCluster.EtcdHosts, clientCert, clientkey)
			if err != nil {
				return err
			}
			if !isEtcdMember {
				if err := services.AddEtcdMember(ctx, etcdHost, kubeCluster.EtcdHosts, clientCert, clientkey); err != nil {
					return err
				}
			}
			etcdHost.ToAddEtcdMember = false
			kubeCluster.setReadyEtcdHosts()

			etcdNodePlanMap := make(map[string]types.ZKENodePlan)
			for _, etcdReadyHost := range kubeCluster.EtcdReadyHosts {
				etcdNodePlanMap[etcdReadyHost.Address] = BuildZKEConfigNodePlan(ctx, kubeCluster, etcdReadyHost, etcdReadyHost.DockerInfo)
			}
			// this will start the newly added etcd node and make sure it started correctly before restarting other node
			// https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/runtime-configuration.md#add-a-new-member
			if err := services.ReloadEtcdCluster(ctx, kubeCluster.EtcdReadyHosts, etcdHost, clientCert, clientkey, currentCluster.PrivateRegistriesMap, etcdNodePlanMap, kubeCluster.Image.Alpine); err != nil {
				return err
			}
		}
		return nil
	}
}

func syncLabels(ctx context.Context, currentCluster, kubeCluster *Cluster) {
	currentHosts := hosts.GetUniqueHostList(currentCluster.EtcdHosts, currentCluster.ControlPlaneHosts, currentCluster.WorkerHosts, currentCluster.EdgeHosts)
	configHosts := hosts.GetUniqueHostList(kubeCluster.EtcdHosts, kubeCluster.ControlPlaneHosts, kubeCluster.WorkerHosts, kubeCluster.EdgeHosts)
	for _, host := range configHosts {
		for _, currentHost := range currentHosts {
			if host.Address == currentHost.Address {
				for k, v := range currentHost.Labels {
					if _, ok := host.Labels[k]; !ok {
						host.ToDelLabels[k] = v
					}
				}
				break
			}
		}
	}
}

func (c *Cluster) setReadyEtcdHosts() {
	c.EtcdReadyHosts = []*hosts.Host{}
	for _, host := range c.EtcdHosts {
		if !host.ToAddEtcdMember {
			c.EtcdReadyHosts = append(c.EtcdReadyHosts, host)
			host.ExistingEtcdCluster = true
		}
	}
}

func cleanControlNode(ctx context.Context, kubeCluster, currentCluster *Cluster, toDeleteHost *hosts.Host) error {

	// if I am deleting a node that's already in the config, it's probably being replaced and I shouldn't remove it  from ks8
	if !hosts.IsNodeInList(toDeleteHost, kubeCluster.ControlPlaneHosts) {
		if err := hosts.DeleteNode(ctx, toDeleteHost, kubeCluster.KubeClient, toDeleteHost.IsWorker); err != nil {
			return fmt.Errorf("Failed to delete controlplane node [%s] from cluster: %v", toDeleteHost.Address, err)
		}
	}
	// attempting to clean services/files on the host
	if err := reconcileHost(ctx, toDeleteHost, false, false, currentCluster.Image.Alpine, currentCluster.DockerDialerFactory, currentCluster.PrivateRegistriesMap, currentCluster.Option.PrefixPath, currentCluster.Option.KubernetesVersion); err != nil {
		log.Warnf(ctx, "[reconcile] Couldn't clean up controlplane node [%s]: %v", toDeleteHost.Address, err)
	}
	return nil
}

func restartComponentsWhenCertChanges(ctx context.Context, currentCluster, kubeCluster *Cluster) error {
	AllCertsMap := map[string]bool{
		pki.KubeAPICertName:            false,
		pki.RequestHeaderCACertName:    false,
		pki.CACertName:                 false,
		pki.ServiceAccountTokenKeyName: false,
		pki.APIProxyClientCertName:     false,
		pki.KubeControllerCertName:     false,
		pki.KubeSchedulerCertName:      false,
		pki.KubeProxyCertName:          false,
		pki.KubeNodeCertName:           false,
	}
	checkCertificateChanges(ctx, currentCluster, kubeCluster, AllCertsMap)
	// check Restart Function
	allHosts := hosts.GetUniqueHostList(kubeCluster.EtcdHosts, kubeCluster.ControlPlaneHosts, kubeCluster.WorkerHosts, kubeCluster.EdgeHosts)
	AllCertsFuncMap := map[string][]services.RestartFunc{
		pki.CACertName:                 []services.RestartFunc{services.RestartKubeAPI, services.RestartKubeController, services.RestartKubelet},
		pki.KubeAPICertName:            []services.RestartFunc{services.RestartKubeAPI, services.RestartKubeController},
		pki.RequestHeaderCACertName:    []services.RestartFunc{services.RestartKubeAPI},
		pki.ServiceAccountTokenKeyName: []services.RestartFunc{services.RestartKubeAPI, services.RestartKubeController},
		pki.APIProxyClientCertName:     []services.RestartFunc{services.RestartKubeAPI},
		pki.KubeControllerCertName:     []services.RestartFunc{services.RestartKubeController},
		pki.KubeSchedulerCertName:      []services.RestartFunc{services.RestartScheduler},
		pki.KubeProxyCertName:          []services.RestartFunc{services.RestartKubeproxy},
		pki.KubeNodeCertName:           []services.RestartFunc{services.RestartKubelet},
	}
	for certName, changed := range AllCertsMap {
		if changed {
			for _, host := range allHosts {
				runRestartFuncs(ctx, AllCertsFuncMap, certName, host)
			}
		}
	}

	for _, host := range kubeCluster.EtcdHosts {
		etcdCertName := pki.GetEtcdCrtName(host.Address)
		certMap := map[string]bool{
			etcdCertName: false,
		}
		checkCertificateChanges(ctx, currentCluster, kubeCluster, certMap)
		if certMap[etcdCertName] || AllCertsMap[pki.CACertName] {
			if err := docker.DoRestartContainer(ctx, host.DClient, services.EtcdContainerName, host.NodeName); err != nil {
				return err
			}
		}
	}
	return nil
}

func runRestartFuncs(ctx context.Context, certFuncMap map[string][]services.RestartFunc, certName string, host *hosts.Host) error {
	for _, restartFunc := range certFuncMap[certName] {
		if err := restartFunc(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

func checkCertificateChanges(ctx context.Context, currentCluster, kubeCluster *Cluster, certMap map[string]bool) {
	for certName := range certMap {
		if currentCluster.Certificates[certName].CertificatePEM != kubeCluster.Certificates[certName].CertificatePEM {
			certMap[certName] = true
			continue
		}
		if !(certName == pki.RequestHeaderCACertName || certName == pki.CACertName) {
			if currentCluster.Certificates[certName].KeyPEM != kubeCluster.Certificates[certName].KeyPEM {
				certMap[certName] = true
			}
		}
	}
}

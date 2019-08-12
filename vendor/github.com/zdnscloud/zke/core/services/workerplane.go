package services

import (
	"context"

	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"github.com/zdnscloud/cement/errgroup"
)

const (
	unschedulableEtcdTaint    = "node-role.kubernetes.io/etcd=true:NoExecute"
	unschedulableControlTaint = "node-role.kubernetes.io/controlplane=true:NoSchedule"
)

func RunWorkerPlane(ctx context.Context, allHosts []*hosts.Host, prsMap map[string]types.PrivateRegistry, workerNodePlanMap map[string]types.ZKENodePlan, certMap map[string]pki.CertificatePKI, alpineImage string) error {
	log.Infof(ctx, "[%s] Building up Worker Plane..", WorkerRole)

	_, err := errgroup.Batch(allHosts, func(h interface{}) (interface{}, error) {
		runHost := h.(*hosts.Host)
		return nil, doDeployWorkerPlaneHost(ctx, runHost, prsMap, workerNodePlanMap[runHost.Address].Processes, certMap, alpineImage)

	})
	if err != nil {
		return err
	}

	log.Infof(ctx, "[%s] Successfully started Worker Plane..", WorkerRole)
	return nil
}

func doDeployWorkerPlaneHost(ctx context.Context, host *hosts.Host, prsMap map[string]types.PrivateRegistry, processMap map[string]types.Process, certMap map[string]pki.CertificatePKI, alpineImage string) error {
	if !host.IsWorker {
		if host.IsEtcd {
			// Add unschedulable taint
			host.ToAddTaints = append(host.ToAddTaints, unschedulableEtcdTaint)
		}
		if host.IsControl {
			// Add unschedulable taint
			host.ToAddTaints = append(host.ToAddTaints, unschedulableControlTaint)
		}
	}
	return doDeployWorkerPlane(ctx, host, prsMap, processMap, certMap, alpineImage)
}

func RemoveWorkerPlane(ctx context.Context, workerHosts []*hosts.Host, force bool) error {
	log.Infof(ctx, "[%s] Tearing down Worker Plane..", WorkerRole)

	_, err := errgroup.Batch(workerHosts, func(h interface{}) (interface{}, error) {
		runHost := h.(*hosts.Host)
		if runHost.IsControl && !force {
			log.Infof(ctx, "[%s] Host [%s] is already a controlplane host, nothing to do.", WorkerRole, runHost.Address)
			return nil, nil
		}
		if err := removeKubelet(ctx, runHost); err != nil {
			return nil, err
		}
		if err := removeKubeproxy(ctx, runHost); err != nil {
			return nil, err
		}
		if err := removeNginxProxy(ctx, runHost); err != nil {
			return nil, err
		}
		if err := removeSidekick(ctx, runHost); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return err
	}

	log.Infof(ctx, "[%s] Successfully tore down Worker Plane..", WorkerRole)

	return nil
}

func RestartWorkerPlane(ctx context.Context, workerHosts []*hosts.Host) error {
	log.Infof(ctx, "[%s] Restarting Worker Plane..", WorkerRole)

	_, err := errgroup.Batch(workerHosts, func(h interface{}) (interface{}, error) {
		runHost := h.(*hosts.Host)
		if err := RestartKubelet(ctx, runHost); err != nil {
			return nil, err
		}
		if err := RestartKubeproxy(ctx, runHost); err != nil {
			return nil, err
		}
		if err := RestartNginxProxy(ctx, runHost); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return err
	}

	log.Infof(ctx, "[%s] Successfully restarted Worker Plane..", WorkerRole)

	return nil
}

func doDeployWorkerPlane(ctx context.Context, host *hosts.Host,
	prsMap map[string]types.PrivateRegistry, processMap map[string]types.Process, certMap map[string]pki.CertificatePKI, alpineImage string) error {
	// run nginx proxy
	if !host.IsControl {
		if err := runNginxProxy(ctx, host, prsMap, processMap[NginxProxyContainerName], alpineImage); err != nil {
			return err
		}
	}
	// run sidekick
	if err := runSidekick(ctx, host, prsMap, processMap[SidekickContainerName]); err != nil {
		return err
	}
	// run kubelet
	if err := runKubelet(ctx, host, prsMap, processMap[KubeletContainerName], certMap, alpineImage); err != nil {
		return err
	}
	return runKubeproxy(ctx, host, prsMap, processMap[KubeproxyContainerName], alpineImage)
}

func copyProcessMap(m map[string]types.Process) map[string]types.Process {
	c := make(map[string]types.Process)
	for k, v := range m {
		c[k] = v
	}
	return c
}

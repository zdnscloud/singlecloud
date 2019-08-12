package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/zdnscloud/zke/core/services"
	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/pkg/util"
)

const (
	SupportedSyncToolsVersion = "0.1.22"
)

func (c *Cluster) SnapshotEtcd(ctx context.Context, snapshotName string) error {
	for _, host := range c.EtcdHosts {
		if err := services.RunEtcdSnapshotSave(ctx, host, c.PrivateRegistriesMap, c.Image.Alpine, snapshotName, true, c.Core.Etcd); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) PrepareBackup(ctx context.Context, snapshotPath string) error {
	// local backup case
	var backupServer *hosts.Host
	// stop etcd on all etcd nodes, we need this because we start the backup server on the same port
	if !isAutoSyncSupported(c.Image.Alpine) {
		log.Warnf(ctx, "Auto local backup sync is not supported. Use `zdnscloud/zke-tools:%s` or up", SupportedSyncToolsVersion)
	} else if c.Core.Etcd.BackupConfig == nil || // legacy zke local backup
		c.Core.Etcd.BackupConfig != nil {
		for _, host := range c.EtcdHosts {
			if err := docker.StopContainer(ctx, host.DClient, host.Address, services.EtcdContainerName); err != nil {
				log.Warnf(ctx, "failed to stop etcd container on host [%s]: %v", host.Address, err)
			}
			if backupServer == nil { // start the download server, only one node should have it!
				if err := services.StartBackupServer(ctx, host, c.PrivateRegistriesMap, c.Image.Alpine, snapshotPath); err != nil {
					log.Warnf(ctx, "failed to start backup server on host [%s]: %v", host.Address, err)
					continue
				}
				backupServer = host
			}
		}
		// start downloading the snapshot
		for _, host := range c.EtcdHosts {
			if backupServer != nil && host.Address == backupServer.Address { // we skip the backup server if it's there
				continue
			}
			if err := services.DownloadEtcdSnapshotFromBackupServer(ctx, host, c.PrivateRegistriesMap, c.Image.Alpine, snapshotPath, backupServer); err != nil {
				return err
			}
		}
		// all good, let's remove the backup server container
		if err := docker.DoRemoveContainer(ctx, backupServer.DClient, services.EtcdServeBackupContainerName, backupServer.Address); err != nil {
			return err
		}
	}
	// this applies to all cases!
	if isEqual := c.etcdSnapshotChecksum(ctx, snapshotPath); !isEqual {
		return fmt.Errorf("etcd snapshots are not consistent")
	}
	return nil
}
func (c *Cluster) RestoreEtcdSnapshot(ctx context.Context, snapshotPath string) error {
	// Start restore process on all etcd hosts
	initCluster := services.GetEtcdInitialCluster(c.EtcdHosts)
	for _, host := range c.EtcdHosts {
		if err := services.RestoreEtcdSnapshot(ctx, host, c.PrivateRegistriesMap, c.Image.Etcd, snapshotPath, initCluster); err != nil {
			return fmt.Errorf("[etcd] Failed to restore etcd snapshot: %v", err)
		}
	}
	return nil
}

func (c *Cluster) etcdSnapshotChecksum(ctx context.Context, snapshotPath string) bool {
	log.Infof(ctx, "[etcd] Checking if all snapshots are identical")
	etcdChecksums := []string{}
	for _, etcdHost := range c.EtcdHosts {
		checksum, err := services.GetEtcdSnapshotChecksum(ctx, etcdHost, c.PrivateRegistriesMap, c.Image.Alpine, snapshotPath)
		if err != nil {
			return false
		}
		etcdChecksums = append(etcdChecksums, checksum)
		log.Infof(ctx, "[etcd] Checksum of etcd snapshot on host [%s] is [%s]", etcdHost.Address, checksum)
	}
	hostChecksum := etcdChecksums[0]
	for _, checksum := range etcdChecksums {
		if checksum != hostChecksum {
			return false
		}
	}
	return true
}

func isAutoSyncSupported(image string) bool {
	v := strings.Split(image, ":")
	last := v[len(v)-1]

	sv, err := util.StrToSemVer(last)
	if err != nil {
		return false
	}

	supported, err := util.StrToSemVer(SupportedSyncToolsVersion)
	if err != nil {
		return false
	}
	if sv.LessThan(*supported) {
		return false
	}
	return true
}

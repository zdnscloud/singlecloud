package main

import (
        "log"
		"flag"

        restresource "github.com/zdnscloud/gorest/resource"
        "github.com/zdnscloud/gorest/resource/schema"
        "github.com/zdnscloud/singlecloud/pkg/handler"
        "github.com/zdnscloud/singlecloud/pkg/types"
)

var (
        Version = restresource.APIVersion{
                Version: "v1",
                Group:   "zcloud.cn",
        }
)

func main() {
		var targetPtah string
		flag.StringVar(&targetPtah, "path", "./doc/", "generate target path")
		flag.Parse()
        schemas := importResource()
        if err := schemas.WriteJsonDocs(&Version, targetPtah); err != nil {
				log.Fatalf("generate resource doc failed. %s", err.Error())
        }
		log.Printf("generate resource doc success")
}

func importResource() *schema.SchemaManager {
        schemas := schema.NewSchemaManager()
        schemas.MustImport(&Version, types.Cluster{}, &handler.ClusterManager{})
        schemas.MustImport(&Version, types.Node{}, &handler.NodeManager{})
        schemas.MustImport(&Version, types.PodNetwork{}, &handler.PodNetworkManager{})
        schemas.MustImport(&Version, types.NodeNetwork{}, &handler.NodeNetworkManager{})
        schemas.MustImport(&Version, types.ServiceNetwork{}, &handler.ServiceManager{})
        schemas.MustImport(&Version, types.BlockDevice{}, &handler.BlockDeviceManager{})
        schemas.MustImport(&Version, types.StorageCluster{}, &handler.StorageClusterManager{})
        schemas.MustImport(&Version, types.Namespace{}, &handler.NamespaceManager{})
        schemas.MustImport(&Version, types.Chart{}, &handler.ChartManager{})
        schemas.MustImport(&Version, types.ConfigMap{}, &handler.ConfigMapManager{})
        schemas.MustImport(&Version, types.CronJob{}, &handler.CronJobManager{})
        schemas.MustImport(&Version, types.DaemonSet{}, &handler.DaemonSetManager{})
        schemas.MustImport(&Version, types.Deployment{}, &handler.DeploymentManager{})
        schemas.MustImport(&Version, types.Ingress{}, &handler.IngressManager{})
        schemas.MustImport(&Version, types.Job{}, &handler.JobManager{})
        schemas.MustImport(&Version, types.LimitRange{}, &handler.LimitRangeManager{})
        schemas.MustImport(&Version, types.PersistentVolumeClaim{}, &handler.PersistentVolumeClaimManager{})
        schemas.MustImport(&Version, types.PersistentVolume{}, &handler.PersistentVolumeManager{})
        schemas.MustImport(&Version, types.ResourceQuota{}, &handler.ResourceQuotaManager{})
        schemas.MustImport(&Version, types.Secret{}, &handler.SecretManager{})
        schemas.MustImport(&Version, types.Service{}, &handler.ServiceManager{})
        schemas.MustImport(&Version, types.StatefulSet{}, &handler.StatefulSetManager{})
        schemas.MustImport(&Version, types.Pod{}, &handler.PodManager{})
        schemas.MustImport(&Version, types.UdpIngress{}, &handler.UDPIngressManager{})
        schemas.MustImport(&Version, types.UserQuota{}, &handler.UserQuotaManager{})
        schemas.MustImport(&Version, types.StorageClass{}, &handler.StorageClassManager{})
        schemas.MustImport(&Version, types.InnerService{}, &handler.InnerServiceManager{})
        schemas.MustImport(&Version, types.OuterService{}, &handler.OuterServiceManager{})
        schemas.MustImport(&Version, types.KubeConfig{}, &handler.KubeConfigManager{})
        schemas.MustImport(&Version, types.Application{}, &handler.ApplicationManager{})
        schemas.MustImport(&Version, types.Monitor{}, &handler.MonitorManager{})
        schemas.MustImport(&Version, types.Registry{}, &handler.RegistryManager{})
        schemas.MustImport(&Version, types.EFK{}, &handler.EFKManager{})
        schemas.MustImport(&Version, types.User{}, &handler.UserManager{})
        return schemas
}

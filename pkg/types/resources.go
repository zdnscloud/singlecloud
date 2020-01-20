package types

import (
	"github.com/zdnscloud/gorest/resource"
)

func Resources() []resource.ResourceKind {
	return []resource.ResourceKind{
		Cluster{},
		Node{},
		PodNetwork{},
		NodeNetwork{},
		ServiceNetwork{},
		BlockDevice{},
		StorageCluster{},
		Namespace{},
		Chart{},
		ConfigMap{},
		CronJob{},
		DaemonSet{},
		Deployment{},
		Ingress{},
		Job{},
		LimitRange{},
		PersistentVolumeClaim{},
		PersistentVolume{},
		ResourceQuota{},
		Secret{},
		Service{},
		StatefulSet{},
		Pod{},
		UDPIngress{},
		StorageClass{},
		InnerService{},
		OuterService{},
		KubeConfig{},
		UserQuota{},
		Application{},
		Monitor{},
		Registry{},
		EFK{},
		User{},
		HorizontalPodAutoscaler{},
		FluentBitConfig{},
		Threshold{},
	}
}

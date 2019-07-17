package zke

import (
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
)

func getZcloudProxyYaml(clusterName string, scAddress string) string {
	return `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: zcloud-proxy
  namespace: zcloud
spec:
  replicas: 1
  selector:
    matchLabels:
      app: zcloud-proxy
  template:
    metadata:
      labels:
        app: zcloud-proxy
    spec:
      containers:
      - args:
        - -server
        - "` + scAddress + `"
        - -cluster
        - "` + clusterName + `"
        command:
        - agent
        image: zdnscloud/zcloud-proxy:v1.0.1
        imagePullPolicy: IfNotPresent
        name: zcloud-proxy
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      securityContext: {}`
}

func deployZcloudProxy(cli client.Client, clusterName, scAddress string) error {
	yaml := getZcloudProxyYaml(clusterName, scAddress)
	return helper.CreateResourceFromYaml(cli, yaml)
}

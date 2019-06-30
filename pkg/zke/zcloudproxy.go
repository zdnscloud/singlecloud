package zke

import (
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
)

func generateZcloudProxyYaml(clusterName string, scURL string) string {
	return `
apiVersion: v1
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
        - "` + scURL + `"
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

func DeployZcloudProxy(cli client.Client, clusterName, scURL string) error {
	yaml := generateZcloudProxyYaml(clusterName, scURL)
	return helper.CreateResourceFromYaml(cli, yaml)
}

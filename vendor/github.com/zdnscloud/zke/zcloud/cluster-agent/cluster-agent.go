package clusteragent

const ClusterAgentTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: zcloud
---
apiVersion: v1
kind: Service
metadata:
  name: cluster-agent
  namespace: zcloud
spec:
  selector:
    app: cluster-agent
  type: ClusterIP
  ports:
  - name: cluster-agent
    port: 80
    targetPort: 8090
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-agent
  namespace: zcloud
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cluster-agent
  template:
    metadata:
      name: cluster-agent
      labels:
        app: cluster-agent
    spec:
      serviceAccount: zcloud-cluster-admin
      containers:
      - name: cluster-agent
        image: {{.Image}}
        ports:
        - name: cluster-agent
          containerPort: 8090
        env:
          - name: CACHE_TIME
            value: "300"
`

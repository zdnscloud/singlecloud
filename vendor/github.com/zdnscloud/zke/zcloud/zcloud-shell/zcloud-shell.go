package zcloudshell

const ZcloudShellTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: zcloud-shell
  namespace: zcloud
spec:
  replicas: 1
  selector:
    matchLabels:
      app: zcloud-shell
  serviceName: zcloud-shell
  template:
    metadata:
      labels:
        app: zcloud-shell
      name: zcloud-shell
    spec:
      containers:
      - image: {{ .ZcloudShellImage}}
        imagePullPolicy: IfNotPresent
        name: zcloud-shell
        securityContext:
          allowPrivilegeEscalation: false
          privileged: false
          runAsUser: 1000
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      serviceAccount: zcloud-cluster-readonly
      serviceAccountName: zcloud-cluster-readonly
`

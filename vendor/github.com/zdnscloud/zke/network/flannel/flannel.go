package flannel

const FlannelTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{ .DeployNamespace }}
{{- if eq .RBACConfig "rbac"}}
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: flannel
subjects:
- kind: ServiceAccount
  name: flannel
  namespace: {{ .DeployNamespace }}
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
{{- end}}
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: kube-flannel-cfg
  namespace: "{{ .DeployNamespace }}"
  labels:
    tier: node
    app: flannel
data:
  cni-conf.json: |
    {
      "name":"cbr0",
      "cniVersion":"0.3.1",
      "plugins":[
        {
          "type":"flannel",
          "delegate":{
            "forceAddress":true,
            "isDefaultGateway":true
          }
        },
        {
          "type":"portmap",
          "capabilities":{
            "portMappings":true
          }
        }
      ]
    }
  net-conf.json: |
    {
      "Network": "{{.ClusterCIDR}}",
      "Backend": {
	{{- if eq .FlannelBackend.Type "vxlan"}}
        "Type": "{{.FlannelBackend.Type}}",
        "Directrouting": {{.FlannelBackend.Directrouting}}
	{{- else}}
        "Type": "{{.FlannelBackend.Type}}"
	{{- end}}
      }
    }
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: kube-flannel
  namespace: "{{ .DeployNamespace }}"
  labels:
    tier: node
    k8s-app: flannel
spec:
  template:
    metadata:
      labels:
        tier: node
        k8s-app: flannel
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                - key: beta.kubernetes.io/os
                  operator: NotIn
                  values:
                    - windows
      serviceAccountName: flannel
      containers:
      - name: kube-flannel
        image: {{.Image}}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 300m
            memory: 500M
          requests:
            cpu: 150m
            memory: 64M
        {{- if .FlannelInterface}}
        command: ["/opt/bin/flanneld","--ip-masq","--kube-subnet-mgr","--iface={{.FlannelInterface}}"]
        {{- else}}
        command: ["/opt/bin/flanneld","--ip-masq","--kube-subnet-mgr"]
        {{- end}}
        securityContext:
          privileged: true
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: run
          mountPath: /run
        - name: cni
          mountPath: /etc/cni/net.d
        - name: flannel-cfg
          mountPath: /etc/kube-flannel/
      - name: install-cni
        image: {{.CNIImage}}
        command: ["/install-cni.sh"]
        env:
        # The CNI network config to install on each node.
        - name: CNI_NETWORK_CONFIG
          valueFrom:
            configMapKeyRef:
              name: kube-flannel-cfg
              key: cni-conf.json
        - name: CNI_CONF_NAME
          value: "10-flannel.conflist"
        volumeMounts:
        - name: cni
          mountPath: /host/etc/cni/net.d
        - name: host-cni-bin
          mountPath: /host/opt/cni/bin/
      hostNetwork: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      - operator: Exists
        effect: NoExecute
      volumes:
        - name: run
          hostPath:
            path: /run
        - name: cni
          hostPath:
            path: /etc/cni/net.d
        - name: flannel-cfg
          configMap:
            name: kube-flannel-cfg
        - name: host-cni-bin
          hostPath:
            path: /opt/cni/bin
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 20%
    type: RollingUpdate
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flannel
  namespace: {{ .DeployNamespace }}`

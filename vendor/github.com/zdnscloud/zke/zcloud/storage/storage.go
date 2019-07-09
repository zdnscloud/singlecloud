package storage

const OperatorTemplate = `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: clusters.storage.zcloud.cn
spec:
  group: storage.zcloud.cn
  names:
    kind: Cluster
    listKind: ClusterList
    plural: clusters
    singular: cluster
  scope: Namespaced
  version: v1
---
{{- if eq .RBACConfig "rbac"}}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: storage-operator
  namespace: zcloud
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: storage-operator-runner
  namespace: zcloud
rules:
  - apiGroups: ["storage.zcloud.cn"]
    resources: ["*"]
    verbs: ["*"]
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csinodeinfos"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["clusterroles", "clusterrolebindings", "roles", "rolebindings"]
    verbs: ["create", "delete", "update", "get", "list", "watch"]
  - apiGroups: [""]
    resources: ["serviceaccounts"]
    verbs: ["create", "delete"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["get", "list" ,"watch" ,"update", "patch", "create"]
  - apiGroups: [""]
    resources: ["persistentvolumes", "persistentvolumeclaims"]
    verbs: ["get" ,"list" ,"watch" ,"update", "create", "delete"]
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["create", "delete"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "get", "list"]
  - apiGroups: ["extensions"]
    resources: ["podsecuritypolicies", "privileged"]
    verbs: ["use"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses", "volumeattachments"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "update", "watch"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["watch", "list"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "create", "delete"]
  - apiGroups: ["apps"]
    resources: ["daemonsets", "statefulsets", "deployments"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "watch", "list", "delete", "update", "create"]
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["get", "watch", "list", "delete", "update", "create"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: storage-operator-role
  namespace: zcloud
subjects:
  - kind: ServiceAccount
    name: storage-operator
    namespace: zcloud
roleRef:
  kind: ClusterRole
  name: storage-operator-runner
  apiGroup: rbac.authorization.k8s.io
---
{{- end}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: storage-operator
  namespace: zcloud
spec:
  replicas: 1
  selector:
    matchLabels:
      app: storage-operator
  template:
    metadata:
      name: storage-operator
      labels:
        app: storage-operator
    spec:
{{- if eq .RBACConfig "rbac"}}
      serviceAccount: storage-operator
{{- end}}
      containers:
      - name: storage-operator
        image: {{.StorageOperatorImage}}
        command: ["/bin/sh", "-c", "/operator -logtostderr"]
`

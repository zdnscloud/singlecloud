package zcloudsa

const SATemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: zcloud
  annotations:
    linkerd.io/inject: disabled
  labels:
    linkerd.io/is-control-plane: "true"
    config.linkerd.io/admission-webhooks: disabled
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: zcloud-cluster-admin
  namespace: zcloud
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: zcloud-cluster-readonly
  namespace: zcloud
---
{{- if eq .RBACConfig "rbac"}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: cluster-readonly
rules:
- apiGroups:
  - '*'
  resources:
  - '*'
  verbs:
  - get
  - list
  - watch
- nonResourceURLs:
  - '*'
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: zcloud-cluster-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: zcloud-cluster-admin
  namespace: zcloud
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: zcloud-cluster-readonly
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-readonly
subjects:
- kind: ServiceAccount
  name: zcloud-cluster-readonly
  namespace: zcloud
{{- end}}
  `

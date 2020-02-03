package servicemesh

const Template = `---
###
### Identity Controller Service RBAC
###
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-identity
  labels:
    linkerd.io/control-plane-component: identity
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: ["authentication.k8s.io"]
  resources: ["tokenreviews"]
  verbs: ["create"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-identity
  labels:
    linkerd.io/control-plane-component: identity
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: linkerd-linkerd-identity
subjects:
- kind: ServiceAccount
  name: linkerd-identity
  namespace: {{ .Namespace }}
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-identity
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: identity
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
###
### Controller RBAC
###
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-controller
  labels:
    linkerd.io/control-plane-component: controller
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: ["extensions", "apps"]
  resources: ["daemonsets", "deployments", "replicasets", "statefulsets"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["extensions", "batch"]
  resources: ["jobs"]
  verbs: ["list" , "get", "watch"]
- apiGroups: [""]
  resources: ["pods", "endpoints", "services", "replicationcontrollers", "namespaces"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["linkerd.io"]
  resources: ["serviceprofiles"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["split.smi-spec.io"]
  resources: ["trafficsplits"]
  verbs: ["list", "get", "watch"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-controller
  labels:
    linkerd.io/control-plane-component: controller
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: linkerd-linkerd-controller
subjects:
- kind: ServiceAccount
  name: linkerd-controller
  namespace: {{ .Namespace }}
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-controller
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: controller
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
###
### Destination Controller Service
###
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-destination
  labels:
    linkerd.io/control-plane-component: destination
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: ["apps"]
  resources: ["replicasets"]
  verbs: ["list", "get", "watch"]
- apiGroups: [""]
  resources: ["pods", "endpoints", "services"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["linkerd.io"]
  resources: ["serviceprofiles"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["split.smi-spec.io"]
  resources: ["trafficsplits"]
  verbs: ["list", "get", "watch"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-destination
  labels:
    linkerd.io/control-plane-component: destination
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: linkerd-linkerd-destination
subjects:
- kind: ServiceAccount
  name: linkerd-destination
  namespace: {{ .Namespace }}
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-destination
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: destination
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
###
### Web RBAC
###
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-web-admin
  labels:
    linkerd.io/control-plane-component: web
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: linkerd-linkerd-tap-admin
subjects:
- kind: ServiceAccount
  name: linkerd-web
  namespace: {{ .Namespace }}
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-web
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: web
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
###
### Service Profile CRD
###
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: serviceprofiles.linkerd.io
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-ns: {{ .Namespace }}
spec:
  group: linkerd.io
  versions:
  - name: v1alpha1
    served: true
    storage: false
  - name: v1alpha2
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: serviceprofiles
    singular: serviceprofile
    kind: ServiceProfile
    shortNames:
    - sp
---
###
### TrafficSplit CRD
### Copied from https://github.com/deislabs/smi-sdk-go/blob/cea7e1e9372304bbb6c74a3f6ca788d9eaa9cc58/crds/split.yaml
###
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: trafficsplits.split.smi-spec.io
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-ns: {{ .Namespace }}
spec:
  group: split.smi-spec.io
  version: v1alpha1
  scope: Namespaced
  names:
    kind: TrafficSplit
    shortNames:
      - ts
    plural: trafficsplits
    singular: trafficsplit
  additionalPrinterColumns:
  - name: Service
    type: string
    description: The apex service of this split.
    JSONPath: .spec.service
---
###
### Prometheus RBAC
###
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-prometheus
  labels:
    linkerd.io/control-plane-component: prometheus
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: [""]
  resources: ["nodes", "nodes/proxy", "pods"]
  verbs: ["get", "list", "watch"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-prometheus
  labels:
    linkerd.io/control-plane-component: prometheus
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: linkerd-linkerd-prometheus
subjects:
- kind: ServiceAccount
  name: linkerd-prometheus
  namespace: {{ .Namespace }}
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-prometheus
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: prometheus
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
###
### Grafana RBAC
###
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-grafana
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: grafana
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
###
### Proxy Injector RBAC
###
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-proxy-injector
  labels:
    linkerd.io/control-plane-component: proxy-injector
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
- apiGroups: [""]
  resources: ["namespaces", "replicationcontrollers"]
  verbs: ["list", "get", "watch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["list", "watch"]
- apiGroups: ["extensions", "apps"]
  resources: ["deployments", "replicasets", "daemonsets", "statefulsets"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["extensions", "batch"]
  resources: ["jobs"]
  verbs: ["list", "get", "watch"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-proxy-injector
  labels:
    linkerd.io/control-plane-component: proxy-injector
    linkerd.io/control-plane-ns: {{ .Namespace }}
subjects:
- kind: ServiceAccount
  name: linkerd-proxy-injector
  namespace: {{ .Namespace }}
  apiGroup: ""
roleRef:
  kind: ClusterRole
  name: linkerd-linkerd-proxy-injector
  apiGroup: rbac.authorization.k8s.io
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-proxy-injector
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: proxy-injector
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
kind: Secret
apiVersion: v1
metadata:
  name: linkerd-proxy-injector-tls
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: proxy-injector
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
type: Opaque
data:
  crt.pem: {{ .LinkerdProxyInjectorTLSCrtPEM }} 
  key.pem: {{ .LinkerdProxyInjectorTLSKeyPEM }} 
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: linkerd-proxy-injector-webhook-config
  labels:
    linkerd.io/control-plane-component: proxy-injector
    linkerd.io/control-plane-ns: {{ .Namespace }}
webhooks:
- name: linkerd-proxy-injector.linkerd.io
  namespaceSelector:
    matchExpressions:
    - key: config.linkerd.io/admission-webhooks
      operator: NotIn
      values:
      - disabled
  clientConfig:
    service:
      name: linkerd-proxy-injector
      namespace: {{ .Namespace }}
      path: "/"
    caBundle: {{ .LinkerdProxyInjectorTLSCrtPEM }} 
  failurePolicy: Ignore
  rules:
  - operations: [ "CREATE" ]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
  sideEffects: None
---
###
### Service Profile Validator RBAC
###
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-sp-validator
  labels:
    linkerd.io/control-plane-component: sp-validator
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["list"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-sp-validator
  labels:
    linkerd.io/control-plane-component: sp-validator
    linkerd.io/control-plane-ns: {{ .Namespace }}
subjects:
- kind: ServiceAccount
  name: linkerd-sp-validator
  namespace: {{ .Namespace }}
  apiGroup: ""
roleRef:
  kind: ClusterRole
  name: linkerd-linkerd-sp-validator
  apiGroup: rbac.authorization.k8s.io
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-sp-validator
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: sp-validator
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
kind: Secret
apiVersion: v1
metadata:
  name: linkerd-sp-validator-tls
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: sp-validator
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
type: Opaque
data:
  crt.pem: {{ .LinkerdSpValidatorTLSCrtPEM }} 
  key.pem: {{ .LinkerdSpValidatorTLSKeyPEM }}
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: linkerd-sp-validator-webhook-config
  labels:
    linkerd.io/control-plane-component: sp-validator
    linkerd.io/control-plane-ns: {{ .Namespace }}
webhooks:
- name: linkerd-sp-validator.linkerd.io
  namespaceSelector:
    matchExpressions:
    - key: config.linkerd.io/admission-webhooks
      operator: NotIn
      values:
      - disabled
  clientConfig:
    service:
      name: linkerd-sp-validator
      namespace: {{ .Namespace }}
      path: "/"
    caBundle: {{ .LinkerdSpValidatorTLSCrtPEM }} 
  failurePolicy: Ignore
  rules:
  - operations: [ "CREATE" , "UPDATE" ]
    apiGroups: ["linkerd.io"]
    apiVersions: ["v1alpha1", "v1alpha2"]
    resources: ["serviceprofiles"]
  sideEffects: None
---
###
### Tap RBAC
###
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-tap
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: [""]
  resources: ["pods", "services", "replicationcontrollers", "namespaces"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["extensions", "apps"]
  resources: ["daemonsets", "deployments", "replicasets", "statefulsets"]
  verbs: ["list", "get", "watch"]
- apiGroups: ["extensions", "batch"]
  resources: ["jobs"]
  verbs: ["list" , "get", "watch"]
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-tap-admin
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: ["tap.linkerd.io"]
  resources: ["*"]
  verbs: ["watch"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: linkerd-linkerd-tap
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: linkerd-linkerd-tap
subjects:
- kind: ServiceAccount
  name: linkerd-tap
  namespace: {{ .Namespace }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: linkerd-linkerd-tap-auth-delegator
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: linkerd-tap
  namespace: {{ .Namespace }}
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: linkerd-tap
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: linkerd-linkerd-tap-auth-reader
  namespace: kube-system
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
- kind: ServiceAccount
  name: linkerd-tap
  namespace: {{ .Namespace }}
---
kind: Secret
apiVersion: v1
metadata:
  name: linkerd-tap-tls
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
type: Opaque
data:
  crt.pem: {{ .LinkerdTapTLSCrtPEM }}
  key.pem: {{ .LinkerdTapTLSKeyPEM }}
---
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  name: v1alpha1.tap.linkerd.io
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
spec:
  group: tap.linkerd.io
  version: v1alpha1
  groupPriorityMinimum: 1000
  versionPriority: 100
  service:
    name: linkerd-tap
    namespace: {{ .Namespace }}
  caBundle: {{ .LinkerdTapTLSCrtPEM }} 
---
###
### Control Plane PSP
###
---
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: linkerd-linkerd-control-plane
  labels:
    linkerd.io/control-plane-ns: {{ .Namespace }}
spec:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  allowedCapabilities:
  - NET_ADMIN
  - NET_RAW
  requiredDropCapabilities:
  - ALL
  hostNetwork: false
  hostIPC: false
  hostPID: false
  seLinux:
    rule: RunAsAny
  runAsUser:
    rule: RunAsAny
  supplementalGroups:
    rule: MustRunAs
    ranges:
    - min: 1
      max: 65535
  fsGroup:
    rule: MustRunAs
    ranges:
    - min: 1
      max: 65535
  volumes:
  - configMap
  - emptyDir
  - secret
  - projected
  - downwardAPI
  - persistentVolumeClaim
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: linkerd-psp
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-ns: {{ .Namespace }}
rules:
- apiGroups: ['policy', 'extensions']
  resources: ['podsecuritypolicies']
  verbs: ['use']
  resourceNames:
  - linkerd-linkerd-control-plane
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: linkerd-psp
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-ns: {{ .Namespace }}
roleRef:
  kind: Role
  name: linkerd-psp
  apiGroup: rbac.authorization.k8s.io
subjects:
- kind: ServiceAccount
  name: linkerd-controller
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-destination
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-grafana
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-identity
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-prometheus
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-proxy-injector
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-sp-validator
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-tap
  namespace: {{ .Namespace }}
- kind: ServiceAccount
  name: linkerd-web
  namespace: {{ .Namespace }}
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: linkerd-config
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: controller
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
data:
  global: |
    {"linkerdNamespace":"{{ .Namespace }}","cniEnabled":false,"version":"stable-2.6.0","identityContext":{"trustDomain": "{{ .ClusterDomain }}","trustAnchorsPem":"{{ .TrustAnchorsPEM | replace "\n" "\\n" }}","issuanceLifetime":"86400s","clockSkewAllowance":"20s"},"autoInjectContext":null,"omitWebhookSideEffects":false,"clusterDomain":"{{ .ClusterDomain }}"}
  proxy: |
    {"proxyImage":{"imageName":"{{ .LinkerdProxyImageName }}","pullPolicy":"IfNotPresent"},"proxyInitImage":{"imageName":"{{ .LinkerdProxyInitImageName }}","pullPolicy":"IfNotPresent"},"controlPort":{"port":4190},"ignoreInboundPorts":[],"ignoreOutboundPorts":[],"inboundPort":{"port":4143},"adminPort":{"port":4191},"outboundPort":{"port":4140},"resource":{"requestCpu":"","requestMemory":"","limitCpu":"","limitMemory":""},"proxyUid":"2102","logLevel":{"level":"warn,linkerd2_proxy=info"},"disableExternalProfiles":true,"proxyVersion":"stable-2.6.0","proxyInitImageVersion":"v1.2.0"}
  install: |
    {"uuid":"{{ .LinkerdConfigInstallUUID }}","cliVersion":"stable-2.6.0","flags":[]}
---
###
### Identity Controller Service
###
---
kind: Secret
apiVersion: v1
metadata:
  name: linkerd-identity-issuer
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: identity
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
    linkerd.io/identity-issuer-expiry: {{ .LinkerdIdentityIssuerExpiry }} 
data:
  crt.pem: {{ .LinkerdIdentityIsserCrtPEM }} 
  key.pem: {{ .LinkerdIdentityIsserKeyPEM }} 
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-identity
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: identity
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: identity
  ports:
  - name: grpc
    port: 8080
    targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: identity
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-identity
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: identity
      linkerd.io/control-plane-ns: {{ .Namespace }}
      linkerd.io/proxy-deployment: linkerd-identity
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: identity
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-identity
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - identity
        - -log-level=info
        image: {{ .LinkerdControllerImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9990
          initialDelaySeconds: 10
        name: identity
        ports:
        - containerPort: 8080
          name: grpc
        - containerPort: 9990
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9990
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/config
          name: config
        - mountPath: /var/run/linkerd/identity/issuer
          name: identity-issuer
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: localhost.:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{.ClusterDomain}}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-identity
      volumes:
      - configMap:
          name: linkerd-config
        name: config
      - name: identity-issuer
        secret:
          secretName: linkerd-identity-issuer
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
###
### Controller
###
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-controller-api
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: controller
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: controller
  ports:
  - name: http
    port: 8085
    targetPort: 8085
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-destination
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: controller
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: controller
  ports:
  - name: grpc
    port: 8086
    targetPort: 8086
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: controller
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-controller
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: controller
      linkerd.io/control-plane-ns: {{ .Namespace }}
      linkerd.io/proxy-deployment: linkerd-controller
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: controller
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-controller
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - public-api
        - -prometheus-url=http://linkerd-prometheus.{{ .Namespace }}.svc.{{ .ClusterDomain }}:9090
        - -destination-addr=linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - -controller-namespace={{ .Namespace }}
        - -log-level=info
        image: {{ .LinkerdControllerImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9995
          initialDelaySeconds: 10
        name: public-api
        ports:
        - containerPort: 8085
          name: http
        - containerPort: 9995
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9995
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/config
          name: config
      - args:
        - destination
        - -addr=:8086
        - -controller-namespace={{ .Namespace }}
        - -enable-h2-upgrade=false
        - -log-level=info
        image: {{ .LinkerdControllerImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9996
          initialDelaySeconds: 10
        name: destination
        ports:
        - containerPort: 8086
          name: grpc
        - containerPort: 9996
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9996
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/config
          name: config
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{.ClusterDomain}}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-controller
      volumes:
      - configMap:
          name: linkerd-config
        name: config
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
###
### Destination Controller Service
###
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-dst
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: destination
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: destination
  ports:
  - name: grpc
    port: 8086
    targetPort: 8086
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: destination
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-destination
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: destination
      linkerd.io/control-plane-ns: {{ .Namespace }}
      linkerd.io/proxy-deployment: linkerd-destination
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: destination
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-destination
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - destination
        - -addr=:8086
        - -controller-namespace={{ .Namespace }}
        - -enable-h2-upgrade=false
        - -log-level=info
        image: {{ .LinkerdControllerImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9996
          initialDelaySeconds: 10
        name: destination
        ports:
        - containerPort: 8086
          name: grpc
        - containerPort: 9996
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9996
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/config
          name: config
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: localhost.:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{.ClusterDomain}}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-destination
      volumes:
      - configMap:
          name: linkerd-config
        name: config
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
###
### Web
###
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-web
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: web
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: web
  ports:
  - name: http
    port: 8084
    targetPort: 8084
  - name: admin-http
    port: 9994
    targetPort: 9994
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: web
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-web
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: web
      linkerd.io/control-plane-ns: {{ .Namespace }}
      linkerd.io/proxy-deployment: linkerd-web
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: web
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-web
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - -api-addr=linkerd-controller-api.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8085
        - -grafana-addr=linkerd-grafana.{{ .Namespace }}.svc.{{ .ClusterDomain }}:3000
        - -controller-namespace={{ .Namespace }}
        - -log-level=info
        image: {{ .LinkerdWebImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9994
          initialDelaySeconds: 10
        name: web
        ports:
        - containerPort: 8084
          name: http
        - containerPort: 9994
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9994
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/config
          name: config
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{.ClusterDomain}}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-web
      volumes:
      - configMap:
          name: linkerd-config
        name: config
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
###
### Prometheus
###
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: linkerd-prometheus-config
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: prometheus
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
data:
  prometheus.yml: |-
    global:
      scrape_interval: 10s
      scrape_timeout: 10s
      evaluation_interval: 10s

    rule_files:
    - /etc/prometheus/*_rules.yml

    scrape_configs:
    - job_name: 'prometheus'
      static_configs:
      - targets: ['localhost:9090']

    - job_name: 'grafana'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ['{{ .Namespace }}']
      relabel_configs:
      - source_labels:
        - __meta_kubernetes_pod_container_name
        action: keep
        regex: ^grafana$

    #  Required for: https://grafana.com/grafana/dashboards/315
    - job_name: 'kubernetes-nodes-cadvisor'
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

      kubernetes_sd_configs:
      - role: node
      relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - target_label: __address__
        replacement: kubernetes.default.svc:443
      - source_labels: [__meta_kubernetes_node_name]
        regex: (.+)
        target_label: __metrics_path__
        replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor
      metric_relabel_configs:
      - source_labels: [__name__]
        regex: '(container|machine)_(cpu|memory|network|fs)_(.+)'
        action: keep
      - source_labels: [__name__]
        regex: 'container_memory_failures_total' # unneeded large metric
        action: drop

    - job_name: 'linkerd-controller'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ['{{ .Namespace }}']
      relabel_configs:
      - source_labels:
        - __meta_kubernetes_pod_label_linkerd_io_control_plane_component
        - __meta_kubernetes_pod_container_port_name
        action: keep
        regex: (.*);admin-http$
      - source_labels: [__meta_kubernetes_pod_container_name]
        action: replace
        target_label: component

    - job_name: 'linkerd-proxy'
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - source_labels:
        - __meta_kubernetes_pod_container_name
        - __meta_kubernetes_pod_container_port_name
        - __meta_kubernetes_pod_label_linkerd_io_control_plane_ns
        action: keep
        regex: ^linkerd-proxy;linkerd-admin;{{ .Namespace }}$
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: namespace
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: pod
      # special case k8s' "job" label, to not interfere with prometheus' "job"
      # label
      # __meta_kubernetes_pod_label_linkerd_io_proxy_job=foo =>
      # k8s_job=foo
      - source_labels: [__meta_kubernetes_pod_label_linkerd_io_proxy_job]
        action: replace
        target_label: k8s_job
      # drop __meta_kubernetes_pod_label_linkerd_io_proxy_job
      - action: labeldrop
        regex: __meta_kubernetes_pod_label_linkerd_io_proxy_job
      # __meta_kubernetes_pod_label_linkerd_io_proxy_deployment=foo =>
      # deployment=foo
      - action: labelmap
        regex: __meta_kubernetes_pod_label_linkerd_io_proxy_(.+)
      # drop all labels that we just made copies of in the previous labelmap
      - action: labeldrop
        regex: __meta_kubernetes_pod_label_linkerd_io_proxy_(.+)
      # __meta_kubernetes_pod_label_linkerd_io_foo=bar =>
      # foo=bar
      - action: labelmap
        regex: __meta_kubernetes_pod_label_linkerd_io_(.+)
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-prometheus
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: prometheus
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: prometheus
  ports:
  - name: admin-http
    port: 9090
    targetPort: 9090
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: prometheus
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-prometheus
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: prometheus
      linkerd.io/control-plane-ns: {{ .Namespace }}
      linkerd.io/proxy-deployment: linkerd-prometheus
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: prometheus
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-prometheus
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - --storage.tsdb.path=/data
        - --storage.tsdb.retention.time=6h
        - --config.file=/etc/prometheus/prometheus.yml
        - --log.level=info
        image: {{ .LinkerdPrometheusImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /-/healthy
            port: 9090
          initialDelaySeconds: 30
          timeoutSeconds: 30
        name: prometheus
        ports:
        - containerPort: 9090
          name: admin-http
        readinessProbe:
          httpGet:
            path: /-/ready
            port: 9090
          initialDelaySeconds: 30
          timeoutSeconds: 30
        securityContext:
          runAsUser: 65534
        volumeMounts:
        - mountPath: /data
          name: data
        - mountPath: /etc/prometheus
          name: prometheus-config
          readOnly: true
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_OUTBOUND_ROUTER_CAPACITY
          value: "10000"
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{.ClusterDomain}}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-prometheus
      volumes:
      - emptyDir: {}
        name: data
      - configMap:
          name: linkerd-prometheus-config
        name: prometheus-config
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
###
### Grafana
###
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: linkerd-grafana-config
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: grafana
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
data:
  grafana.ini: |-
    instance_name = linkerd-grafana

    [server]
    root_url = %(protocol)s://%(domain)s:/grafana/

    [auth]
    disable_login_form = true

    [auth.anonymous]
    enabled = true
    org_role = Editor

    [auth.basic]
    enabled = false

    [analytics]
    check_for_updates = false

    [panels]
    disable_sanitize_html = true

  datasources.yaml: |-
    apiVersion: 1
    datasources:
    - name: prometheus
      type: prometheus
      access: proxy
      orgId: 1
      url: http://linkerd-prometheus.{{ .Namespace }}.svc.{{ .ClusterDomain }}:9090
      isDefault: true
      jsonData:
        timeInterval: "5s"
      version: 1
      editable: true

  dashboards.yaml: |-
    apiVersion: 1
    providers:
    - name: 'default'
      orgId: 1
      folder: ''
      type: file
      disableDeletion: true
      editable: true
      options:
        path: /var/lib/grafana/dashboards
        homeDashboardId: linkerd-top-line
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-grafana
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: grafana
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: grafana
  ports:
  - name: http
    port: 3000
    targetPort: 3000
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: grafana
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-grafana
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: grafana
      linkerd.io/control-plane-ns: {{ .Namespace }}
      linkerd.io/proxy-deployment: linkerd-grafana
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: grafana
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-grafana
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - env:
        - name: GF_PATHS_DATA
          value: /data
        image: {{ .LinkerdGrafanaImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /api/health
            port: 3000
          initialDelaySeconds: 30
        name: grafana
        ports:
        - containerPort: 3000
          name: http
        readinessProbe:
          httpGet:
            path: /api/health
            port: 3000
        securityContext:
          runAsUser: 472
        volumeMounts:
        - mountPath: /data
          name: data
        - mountPath: /etc/grafana
          name: grafana-config
          readOnly: true
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{ .ClusterDomain }}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-grafana
      volumes:
      - emptyDir: {}
        name: data
      - configMap:
          items:
          - key: grafana.ini
            path: grafana.ini
          - key: datasources.yaml
            path: provisioning/datasources/datasources.yaml
          - key: dashboards.yaml
            path: provisioning/dashboards/dashboards.yaml
          name: linkerd-grafana-config
        name: grafana-config
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
###
### Proxy Injector
###
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: proxy-injector
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-proxy-injector
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: proxy-injector
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: proxy-injector
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-proxy-injector
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - proxy-injector
        - -log-level=info
        image: {{ .LinkerdControllerImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9995
          initialDelaySeconds: 10
        name: proxy-injector
        ports:
        - containerPort: 8443
          name: proxy-injector
        - containerPort: 9995
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9995
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/config
          name: config
        - mountPath: /var/run/linkerd/tls
          name: tls
          readOnly: true
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{ .ClusterDomain }}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-proxy-injector
      volumes:
      - configMap:
          name: linkerd-config
        name: config
      - name: tls
        secret:
          secretName: linkerd-proxy-injector-tls
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-proxy-injector
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: proxy-injector
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: proxy-injector
  ports:
  - name: proxy-injector
    port: 443
    targetPort: proxy-injector
---
###
### Service Profile Validator
###
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-sp-validator
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: sp-validator
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: sp-validator
  ports:
  - name: sp-validator
    port: 443
    targetPort: sp-validator
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: sp-validator
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-sp-validator
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: sp-validator
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: sp-validator
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-sp-validator
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - sp-validator
        - -log-level=info
        image: {{ .LinkerdControllerImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9997
          initialDelaySeconds: 10
        name: sp-validator
        ports:
        - containerPort: 8443
          name: sp-validator
        - containerPort: 9997
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9997
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/tls
          name: tls
          readOnly: true
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{ .ClusterDomain }}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-sp-validator
      volumes:
      - name: tls
        secret:
          secretName: linkerd-sp-validator-tls
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
---
###
### Tap
###
---
kind: Service
apiVersion: v1
metadata:
  name: linkerd-tap
  namespace: {{ .Namespace }}
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
spec:
  type: ClusterIP
  selector:
    linkerd.io/control-plane-component: tap
  ports:
  - name: grpc
    port: 8088
    targetPort: 8088
  - name: apiserver
    port: 443
    targetPort: apiserver
---
kind: Deployment
apiVersion: apps/v1
metadata:
  annotations:
    linkerd.io/created-by: linkerd/cli stable-2.6.0
  labels:
    linkerd.io/control-plane-component: tap
    linkerd.io/control-plane-ns: {{ .Namespace }}
  name: linkerd-tap
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      linkerd.io/control-plane-component: tap
      linkerd.io/control-plane-ns: {{ .Namespace }}
      linkerd.io/proxy-deployment: linkerd-tap
  template:
    metadata:
      annotations:
        linkerd.io/created-by: linkerd/cli stable-2.6.0
        linkerd.io/identity-mode: default
        linkerd.io/proxy-version: stable-2.6.0
      labels:
        linkerd.io/control-plane-component: tap
        linkerd.io/control-plane-ns: {{ .Namespace }}
        linkerd.io/proxy-deployment: linkerd-tap
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - args:
        - tap
        - -controller-namespace={{ .Namespace }}
        - -log-level=info
        image: {{ .LinkerdControllerImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /ping
            port: 9998
          initialDelaySeconds: 10
        name: tap
        ports:
        - containerPort: 8088
          name: grpc
        - containerPort: 8089
          name: apiserver
        - containerPort: 9998
          name: admin-http
        readinessProbe:
          failureThreshold: 7
          httpGet:
            path: /ready
            port: 9998
        securityContext:
          runAsUser: 2103
        volumeMounts:
        - mountPath: /var/run/linkerd/tls
          name: tls
          readOnly: true
        - mountPath: /var/run/linkerd/config
          name: config
      - env:
        - name: LINKERD2_PROXY_LOG
          value: warn,linkerd2_proxy=info
        - name: LINKERD2_PROXY_DESTINATION_SVC_ADDR
          value: linkerd-dst.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8086
        - name: LINKERD2_PROXY_CONTROL_LISTEN_ADDR
          value: 0.0.0.0:4190
        - name: LINKERD2_PROXY_ADMIN_LISTEN_ADDR
          value: 0.0.0.0:4191
        - name: LINKERD2_PROXY_OUTBOUND_LISTEN_ADDR
          value: 127.0.0.1:4140
        - name: LINKERD2_PROXY_INBOUND_LISTEN_ADDR
          value: 0.0.0.0:4143
        - name: LINKERD2_PROXY_DESTINATION_GET_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES
          value: svc.{{ .ClusterDomain }}.
        - name: LINKERD2_PROXY_INBOUND_ACCEPT_KEEPALIVE
          value: 10000ms
        - name: LINKERD2_PROXY_OUTBOUND_CONNECT_KEEPALIVE
          value: 10000ms
        - name: _pod_ns
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: LINKERD2_PROXY_DESTINATION_CONTEXT
          value: ns:$(_pod_ns)
        - name: LINKERD2_PROXY_IDENTITY_DIR
          value: /var/run/linkerd/identity/end-entity
        - name: LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS
          value: |{{ .TrustAnchorsPEM | nindent 12 | trimPrefix (repeat 7 " ")}}
        - name: LINKERD2_PROXY_IDENTITY_TOKEN_FILE
          value: /var/run/secrets/kubernetes.io/serviceaccount/token
        - name: LINKERD2_PROXY_IDENTITY_SVC_ADDR
          value: linkerd-identity.{{ .Namespace }}.svc.{{ .ClusterDomain }}:8080
        - name: _pod_sa
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: _l5d_ns
          value: {{ .Namespace }}
        - name: _l5d_trustdomain
          value: {{ .ClusterDomain }}
        - name: LINKERD2_PROXY_IDENTITY_LOCAL_NAME
          value: $(_pod_sa).$(_pod_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_IDENTITY_SVC_NAME
          value: linkerd-identity.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_DESTINATION_SVC_NAME
          value: linkerd-destination.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        - name: LINKERD2_PROXY_TAP_SVC_NAME
          value: linkerd-tap.$(_l5d_ns).serviceaccount.identity.$(_l5d_ns).$(_l5d_trustdomain)
        image: {{ .LinkerdProxyImage }}
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /metrics
            port: 4191
          initialDelaySeconds: 10
        name: linkerd-proxy
        ports:
        - containerPort: 4143
          name: linkerd-proxy
        - containerPort: 4191
          name: linkerd-admin
        readinessProbe:
          httpGet:
            path: /ready
            port: 4191
          initialDelaySeconds: 2
        resources:
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 2102
        terminationMessagePolicy: FallbackToLogsOnError
        volumeMounts:
        - mountPath: /var/run/linkerd/identity/end-entity
          name: linkerd-identity-end-entity
      initContainers:
      - args:
        - --incoming-proxy-port
        - "4143"
        - --outgoing-proxy-port
        - "4140"
        - --proxy-uid
        - "2102"
        - --inbound-ports-to-ignore
        - 4190,4191
        - --outbound-ports-to-ignore
        - "443"
        image: {{ .LinkerdProxyInitImage }}
        imagePullPolicy: IfNotPresent
        name: linkerd-init
        resources:
          limits:
            cpu: "100m"
            memory: "50Mi"
          requests:
            cpu: "10m"
            memory: "10Mi"
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: false
          runAsUser: 0
        terminationMessagePolicy: FallbackToLogsOnError
      serviceAccountName: linkerd-tap
      volumes:
      - configMap:
          name: linkerd-config
        name: config
      - emptyDir:
          medium: Memory
        name: linkerd-identity-end-entity
      - name: tls
        secret:
          secretName: linkerd-tap-tls
---
`

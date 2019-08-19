package prometheus

const EtcdClientCertTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: zcloud
---
apiVersion: v1
kind: Secret
metadata:
  labels:
    app: zcloud-monitor
  namespace: zcloud
  name: zcloud-monitor-etcd-client-cert
data:
  etcd-ca: {{ .EtcdClientCa }}
  etcd-client: {{ .EtcdClientCert }}
  etcd-client-key: {{ .EtcdClientKey }}
`

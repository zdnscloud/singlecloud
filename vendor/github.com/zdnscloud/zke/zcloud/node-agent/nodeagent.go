package nodeagent

const NodeAgentTemplate = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-agent
  namespace: zcloud
  labels:
    app: node-agent
spec:
  selector:
    matchLabels:
      app: node-agent
  template:
    metadata:
      labels:
        app: node-agent
    spec:
      tolerations:
      - operator: Exists
        effect: NoSchedule
      - operator: Exists
        effect: NoExecute
      containers:
      - name: node-agent
        image: {{.Image}}
        command: ["/bin/sh", "-c", "/node-agent -listen $(POD_IP):$(SVC_PORT) -node $(NODE_NAME)"]
        env:
          - name: SVC_PORT
            value: "{{.NodeAgentPort}}"    
          - name: NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          - name: POD_IP
            valueFrom:
              fieldRef:
                fieldPath: status.podIP
        securityContext:
          privileged: true
        volumeMounts:
          - mountPath: /var/lib
            name: lib
          - mountPath: /dev
            name: host-dev
      - name: orphaned-clean
        image: {{.Image}}
        command: ["/bin/sh", "-c", "/orphaned-clean"]
        securityContext:
          privileged: true
        volumeMounts:
          - mountPath: /var/lib
            name: lib
          - mountPath: /var/run/docker.sock
            name: docker
      volumes:
        - name: lib
          hostPath:
            path: /var/lib
        - name: docker
          hostPath:
            path: /var/run/docker.sock
            type: Socket
        - name: host-dev
          hostPath:
            path: /dev`

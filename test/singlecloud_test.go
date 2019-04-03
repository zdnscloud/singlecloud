package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	ut "github.com/zdnscloud/cement/unittest"

	restTypes "github.com/zdnscloud/gorest/types"

	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/server"
)

const (
	addr               = "0.0.0.0:1234"
	clusterName        = "test-cluster1"
	namespaceName      = "test-namespace1"
	deploymentName     = "test-deployment1"
	containerName      = "test-containter1"
	configMapName      = "test-configmap1"
	configMountPath    = "/etc/config"
	secretName         = "test-secret1"
	secretMountPath    = "/etc/secret"
	secretDataName     = "test-secret-dataname1"
	secretData         = "emRucw=="
	ingressName        = "test-ingress1"
	ingressPath        = "/etc/ingress"
	exposedPortName    = "test-port1"
	exposedPort        = 22222
	exposedProtocol    = "tcp"
	exposedServiceType = "clusterip"
	configMapDataName  = "test-cm-dataname1"
	configMapData      = "test-cm-data1"
	jobName            = "test-job1"
	cronjobName        = "test-cronjob1"
	cronjobSchedule    = "*/1 * * * *"
	restartPolicy      = "onfailure"
)

func runTestServer() {
	if err := logger.InitLogger(); err != nil {
		panic("init logger failed:" + err.Error())
	}

	server, err := server.NewServer()
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	if err := server.Run(addr); err != nil {
		panic("server run failed:" + err.Error())
	}
}

func importTestCluster() error {
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user failed:%s", err.Error())
	}

	k8sconfig := filepath.Join(usr.HomeDir, ".kube", "config")
	f, err := os.Open(k8sconfig)
	if err != nil {
		return fmt.Errorf("open %s failed:%s", k8sconfig, err.Error())
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read %s failed:%s", k8sconfig, err.Error())
	}

	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters", addr)
	params := map[string]interface{}{
		"name":  clusterName,
		"yaml_": string(data),
	}

	return sendRequest("POST", url, getBodyFromMap(params), nil)
}

func TestSingleCloud(t *testing.T) {
	go runTestServer()
	time.Sleep(1 * time.Second)
	testCluster(t)
	testNamespace(t)
	testConfigMap(t)
	testSecret(t)
	testDeployment(t)
	testJob(t)
	testCronJob(t)
	testClearCluster(t)
}

func testCluster(t *testing.T) {
	err := importTestCluster()
	ut.Equal(t, err, nil)

	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters", addr)
	var collection restTypes.Collection
	err = sendRequest("GET", url, nil, &collection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 1)
	c := sliceData.Index(0).Interface()
	object, ok := c.(map[string]interface{})
	ut.Equal(t, ok, true)
	ut.Equal(t, object["name"], clusterName)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s", addr, clusterName)
	var cluster types.Cluster
	err = sendRequest("GET", url, nil, &cluster)
	ut.Equal(t, err, nil)
	ut.Equal(t, cluster.Name, clusterName)
}

func testNamespace(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces", addr, clusterName)
	var oldcollection restTypes.Collection
	err := sendRequest("GET", url, nil, &oldcollection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(oldcollection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	existNamespaceNum := sliceData.Len()

	params := map[string]interface{}{
		"name": namespaceName,
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	var collection restTypes.Collection
	err = sendRequest("GET", url, nil, &collection)
	ut.Equal(t, err, nil)
	sliceData = reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), existNamespaceNum+1)
	exits := false
	for i := 0; i < sliceData.Len(); i++ {
		c := sliceData.Index(i).Interface()
		object, ok := c.(map[string]interface{})
		ut.Equal(t, ok, true)
		if object["name"] == namespaceName {
			exits = true
		}
	}
	ut.Equal(t, exits, true)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s", addr, clusterName, namespaceName)
	var namespace types.Namespace
	err = sendRequest("GET", url, nil, &namespace)
	ut.Equal(t, err, nil)
	ut.Equal(t, namespace.Name, namespaceName)
}

func testConfigMap(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/configmaps", addr, clusterName, namespaceName)
	var oldcollection restTypes.Collection
	err := sendRequest("GET", url, nil, &oldcollection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(oldcollection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 0)

	params := map[string]interface{}{
		"name": configMapName,
		"configs": []types.Config{
			types.Config{
				Name: configMapDataName,
				Data: configMapData,
			},
		},
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	var collection restTypes.Collection
	err = sendRequest("GET", url, nil, &collection)
	ut.Equal(t, err, nil)
	sliceData = reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 1)
	c := sliceData.Index(0).Interface()
	object, ok := c.(map[string]interface{})
	ut.Equal(t, ok, true)
	ut.Equal(t, object["name"], configMapName)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/configmaps/%s",
		addr, clusterName, namespaceName, configMapName)
	var configMap types.ConfigMap
	err = sendRequest("GET", url, nil, &configMap)
	ut.Equal(t, err, nil)
	ut.Equal(t, configMap.Name, configMapName)
	ut.Equal(t, configMap.Configs[0].Name, configMapDataName)
}

func testSecret(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/secrets", addr, clusterName, namespaceName)
	var oldcollection restTypes.Collection
	err := sendRequest("GET", url, nil, &oldcollection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(oldcollection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	//default-token
	ut.Equal(t, sliceData.Len(), 1)

	params := map[string]interface{}{
		"name": secretName,
		"data": map[string]string{
			secretDataName: secretData,
		},
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	var collection restTypes.Collection
	err = sendRequest("GET", url, nil, &collection)
	ut.Equal(t, err, nil)
	sliceData = reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 2)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/secrets/%s",
		addr, clusterName, namespaceName, secretName)
	var secret types.Secret
	err = sendRequest("GET", url, nil, &secret)
	ut.Equal(t, err, nil)
	ut.Equal(t, secret.Name, secretName)
	ut.Equal(t, secret.Data[secretDataName], secretData)
}

func testDeployment(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/deployments", addr, clusterName, namespaceName)
	var oldcollection restTypes.Collection
	err := sendRequest("GET", url, nil, &oldcollection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(oldcollection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 0)

	containers := []types.Container{
		types.Container{
			Name:       containerName,
			Image:      "redis",
			Command:    []string{"ls", "-l"},
			Args:       []string{"/"},
			ConfigName: configMapName,
			MountPath:  configMountPath,
			SecretName: secretName,
			SecretPath: secretMountPath,
			Env: map[string]string{
				"TESTENV1": "testenv1",
			},
			ExposedPorts: []types.DeploymentPort{
				types.DeploymentPort{
					Name:     exposedPortName,
					Port:     exposedPort,
					Protocol: exposedProtocol,
				},
			},
		},
	}

	advancedOptions := types.AdvancedOptions{
		ExposedServiceType: exposedServiceType,
		ExposedServices: []types.ExposedService{
			types.ExposedService{
				Name:              exposedPortName,
				Port:              exposedPort,
				Protocol:          exposedProtocol,
				ServicePort:       exposedPort,
				AutoCreateIngress: true,
				IngressDomainName: ingressName,
				IngressPath:       ingressPath,
			},
		},
	}

	params := map[string]interface{}{
		"name":            deploymentName,
		"replicas":        2,
		"containers":      containers,
		"advancedOptions": advancedOptions,
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	var collection restTypes.Collection
	err = sendRequest("GET", url, nil, &collection)
	ut.Equal(t, err, nil)
	sliceData = reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 1)
	c := sliceData.Index(0).Interface()
	object, ok := c.(map[string]interface{})
	ut.Equal(t, ok, true)
	ut.Equal(t, object["name"], deploymentName)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/deployments/%s",
		addr, clusterName, namespaceName, deploymentName)
	var deploy types.Deployment
	err = sendRequest("GET", url, nil, &deploy)
	ut.Equal(t, err, nil)
	ut.Equal(t, deploy.Name, deploymentName)
	ut.Equal(t, deploy.Containers[0].Name, containerName)
	ut.Equal(t, deploy.Containers[0].ConfigName, configMapName)
	ut.Equal(t, deploy.Containers[0].SecretName, secretName)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/services/%s",
		addr, clusterName, namespaceName, deploymentName)
	var service types.Service
	err = sendRequest("GET", url, nil, &service)
	ut.Equal(t, err, nil)
	ut.Equal(t, service.Name, deploymentName)
	ut.Equal(t, service.ServiceType, exposedServiceType)
	ut.Equal(t, service.ExposedPorts[0].Name, exposedPortName)
	ut.Equal(t, service.ExposedPorts[0].Port, exposedPort)
	ut.Equal(t, service.ExposedPorts[0].Protocol, exposedProtocol)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/ingresses/%s",
		addr, clusterName, namespaceName, deploymentName)
	var ingress types.Ingress
	err = sendRequest("GET", url, nil, &ingress)
	ut.Equal(t, err, nil)
	ut.Equal(t, ingress.Name, deploymentName)
	ut.Equal(t, ingress.Rules[0].Host, ingressName)
	ut.Equal(t, ingress.Rules[0].Paths[0].Path, ingressPath)
	ut.Equal(t, ingress.Rules[0].Paths[0].ServicePort, exposedPort)
	ut.Equal(t, ingress.Rules[0].Paths[0].ServiceName, deploymentName)
}

func testCronJob(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/cronjobs", addr, clusterName, namespaceName)
	var oldcollection restTypes.Collection
	err := sendRequest("GET", url, nil, &oldcollection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(oldcollection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 0)

	params := map[string]interface{}{
		"name":          cronjobName,
		"schedule":      cronjobSchedule,
		"restartPolicy": restartPolicy,
		"containers": []types.Container{
			types.Container{
				Name:  containerName,
				Image: "redis",
			},
		},
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	var collection restTypes.Collection
	err = sendRequest("GET", url, nil, &collection)
	ut.Equal(t, err, nil)
	sliceData = reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 1)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/cronjobs/%s",
		addr, clusterName, namespaceName, cronjobName)
	var cronjob types.CronJob
	err = sendRequest("GET", url, nil, &cronjob)
	ut.Equal(t, err, nil)
	ut.Equal(t, cronjob.Name, cronjobName)
	ut.Equal(t, cronjob.Schedule, cronjobSchedule)
	ut.Equal(t, cronjob.Containers[0].Name, containerName)
}

func testJob(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/jobs", addr, clusterName, namespaceName)
	var oldcollection restTypes.Collection
	err := sendRequest("GET", url, nil, &oldcollection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(oldcollection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 0)

	params := map[string]interface{}{
		"name":          jobName,
		"restartPolicy": restartPolicy,
		"containers": []types.Container{
			types.Container{
				Name:  containerName,
				Image: "redis",
			},
		},
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	var collection restTypes.Collection
	err = sendRequest("GET", url, nil, &collection)
	ut.Equal(t, err, nil)
	sliceData = reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 1)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/jobs/%s",
		addr, clusterName, namespaceName, jobName)
	var job types.Job
	err = sendRequest("GET", url, nil, &job)
	ut.Equal(t, err, nil)
	ut.Equal(t, job.Name, jobName)
	ut.Equal(t, job.Containers[0].Name, containerName)
}

func testClearCluster(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/deployments/%s",
		addr, clusterName, namespaceName, deploymentName)
	err := sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/configmaps/%s",
		addr, clusterName, namespaceName, configMapName)
	err = sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/secrets/%s",
		addr, clusterName, namespaceName, secretName)
	err = sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/jobs/%s",
		addr, clusterName, namespaceName, jobName)
	err = sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/cronjobs/%s",
		addr, clusterName, namespaceName, cronjobName)
	err = sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s", addr, clusterName, namespaceName)
	err = sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)
}

func getBodyFromMap(params map[string]interface{}) io.Reader {
	requestBody, _ := json.Marshal(params)
	return bytes.NewBuffer(requestBody)
}

func sendRequest(method, url string, reqBody io.Reader, result interface{}) error {
	req, _ := http.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		err := json.Unmarshal(body, result)
		return err
	case http.StatusCreated:
		fallthrough
	case http.StatusNoContent:
		return nil
	default:
		errInfo := struct {
			Message string `json:"message"`
		}{}
		json.Unmarshal(body, &errInfo)
		return fmt.Errorf("%s %s failed: %s", method, url, errInfo.Message)
	}
}

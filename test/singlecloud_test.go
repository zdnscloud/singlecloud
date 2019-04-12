package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
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
	clusterName        = "sc-test-cluster1"
	namespaceName      = "sc-test-namespace1"
	deploymentName     = "sc-test-deployment1"
	containerName      = "sc-test-containter1"
	configMapName      = "sc-test-configmap1"
	configMountPath    = "/etc/scconfig"
	secretName         = "sc-test-secret1"
	secretMountPath    = "/etc/scsecret"
	secretDataName     = "sc-test-secret-dataname1"
	secretData         = "emRucw=="
	ingressName        = "sc-test-ingress1"
	ingressPath        = "/etc/scingress"
	exposedPortName    = "sc-test-port1"
	exposedPort        = 22222
	exposedProtocol    = "tcp"
	exposedServiceType = "clusterip"
	configMapDataName  = "sc-test-cm-dataname1"
	configMapData      = "sc-test-cm-data1"
	jobName            = "sc-test-job1"
	cronjobName        = "sc-test-cronjob1"
	cronjobSchedule    = "*/1 * * * *"
	restartPolicy      = "onfailure"
	adminUserName      = "admin"
	adminPassword      = "zdns"
	limitRangeName     = "sc-test-limitrange1"
	resourceQuotaName  = "sc-test-resourcequota1"
	userName           = "sc-test-username1"
	userPass           = "sc-test-userpass1"
)

var gToken string

type Token struct {
	Token string `json:"token"`
}

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

func genPassword(pass string) string {
	pwHash := sha1.Sum([]byte(pass))
	return hex.EncodeToString(pwHash[:])
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

	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/users/admin?action=login", addr)
	params := map[string]interface{}{
		"user":     adminUserName,
		"password": genPassword(adminPassword),
	}
	var token Token
	err = sendRequest("POST", url, getBodyFromMap(params), &token)
	if err != nil {
		return fmt.Errorf("login failed:%s", err.Error())
	}
	gToken = token.Token

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters", addr)
	params = map[string]interface{}{
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
	testLimitRange(t)
	testResourceQuota(t)
	testUser(t)
	testClearCluster(t)
}

func testCluster(t *testing.T) {
	err := importTestCluster()
	ut.Equal(t, err, nil)

	existClusterNum, err := getCollectionNum(fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters", addr))
	ut.Equal(t, err, nil)
	ut.Equal(t, existClusterNum, 1)

	var cluster types.Cluster
	err = sendRequest("GET", fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s", addr, clusterName), nil, &cluster)
	ut.Equal(t, err, nil)
	ut.Equal(t, cluster.Name, clusterName)
}

func testNamespace(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces", addr, clusterName)
	existNamespaceNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)

	params := map[string]interface{}{
		"name": namespaceName,
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	newNamespaceNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newNamespaceNum, existNamespaceNum+1)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s", addr, clusterName, namespaceName)
	var namespace types.Namespace
	err = sendRequest("GET", url, nil, &namespace)
	ut.Equal(t, err, nil)
	ut.Equal(t, namespace.Name, namespaceName)
}

func testConfigMap(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/configmaps", addr, clusterName, namespaceName)
	existConfigMapNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, existConfigMapNum, 0)

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

	newConfigMapNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newConfigMapNum, 1)

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
	existSecretNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	//default-token
	ut.Equal(t, existSecretNum, 1)

	params := map[string]interface{}{
		"name": secretName,
		"data": map[string]string{
			secretDataName: secretData,
		},
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	newSecretNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newSecretNum, 2)

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
	existDeploymentNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, existDeploymentNum, 0)

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

	newDeploymentNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newDeploymentNum, 1)

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
	existCronJobNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, existCronJobNum, 0)

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

	newCronJobNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newCronJobNum, 1)

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
	existJobNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, existJobNum, 0)

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

	newJobNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newJobNum, 1)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/jobs/%s",
		addr, clusterName, namespaceName, jobName)
	var job types.Job
	err = sendRequest("GET", url, nil, &job)
	ut.Equal(t, err, nil)
	ut.Equal(t, job.Name, jobName)
	ut.Equal(t, job.Containers[0].Name, containerName)
}

func testLimitRange(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/limitranges",
		addr, clusterName, namespaceName)
	existLimitsNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, existLimitsNum, 0)

	params := map[string]interface{}{
		"name": limitRangeName,
		"max": map[string]string{
			"cpu":    "200m",
			"memory": "200Ki",
		},
		"min": map[string]string{
			"cpu":    "100m",
			"memory": "100Ki",
		},
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	newLimitsNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newLimitsNum, 1)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/limitranges/%s",
		addr, clusterName, namespaceName, limitRangeName)
	var limit types.LimitRange
	err = sendRequest("GET", url, nil, &limit)
	ut.Equal(t, err, nil)
	ut.Equal(t, limit.Name, limitRangeName)
	ut.Equal(t, limit.Max["cpu"], "200m")
	ut.Equal(t, limit.Max["memory"], "200Ki")
	ut.Equal(t, limit.Min["cpu"], "100m")
	ut.Equal(t, limit.Min["memory"], "100Ki")
}

func testResourceQuota(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/resourcequotas",
		addr, clusterName, namespaceName)
	existQuotasNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, existQuotasNum, 0)

	params := map[string]interface{}{
		"name": resourceQuotaName,
		"limits": map[string]string{
			"requests.cpu":    "200m",
			"requests.memory": "200Ki",
			"limits.cpu":      "200m",
			"limits.memory":   "200Ki",
		},
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	newQuotasNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newQuotasNum, 1)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/resourcequotas/%s",
		addr, clusterName, namespaceName, resourceQuotaName)
	var quota types.ResourceQuota
	err = sendRequest("GET", url, nil, &quota)
	ut.Equal(t, err, nil)
	ut.Equal(t, quota.Name, resourceQuotaName)
	ut.Equal(t, quota.Limits["limits.cpu"], "200m")
	ut.Equal(t, quota.Limits["limits.memory"], "200Ki")
	ut.Equal(t, quota.Limits["requests.cpu"], "200m")
	ut.Equal(t, quota.Limits["requests.memory"], "200Ki")
}

func testUser(t *testing.T) {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/users", addr)
	existUserNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	//admin
	ut.Equal(t, existUserNum, 1)

	params := map[string]interface{}{
		"name":     userName,
		"password": genPassword(userPass),
	}
	err = sendRequest("POST", url, getBodyFromMap(params), nil)
	ut.Equal(t, err, nil)

	newUserNum, err := getCollectionNum(url)
	ut.Equal(t, err, nil)
	ut.Equal(t, newUserNum, 2)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/users/%s", addr, userName)
	var user types.User
	err = sendRequest("GET", url, nil, &user)
	ut.Equal(t, err, nil)
	ut.Equal(t, user.Name, userName)

	params = map[string]interface{}{
		"user":     userName,
		"password": genPassword(userPass),
	}
	var token Token
	err = sendRequest("POST", url, getBodyFromMap(params), &token)
	if err != nil {
		return fmt.Errorf("login failed:%s", err.Error())
	}
	adminToken = gToken
	gToken = token.Token
	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/users/%s", addr, userName)
	var user types.User
	err = sendRequest("GET", url, nil, &user)
	ut.Equal(t, err, nil)
	ut.Equal(t, user.Name, userName)
	gToken = adminToken
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

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/limitranges/%s",
		addr, clusterName, namespaceName, limitRangeName)
	err = sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/%s/resourcequotas/%s",
		addr, clusterName, namespaceName, resourceQuotaName)
	err = sendRequest("DELETE", url, nil, nil)
	ut.Equal(t, err, nil)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/users/%s", addr, userName)
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
	req.Header.Add("Authorization", "Bearer "+gToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
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

func getCollectionNum(url string) (int, error) {
	var oldcollection restTypes.Collection
	err := sendRequest("GET", url, nil, &oldcollection)
	if err != nil {
		return 0, err
	}
	sliceData := reflect.ValueOf(oldcollection.Data)
	if sliceData.Kind() != reflect.Slice {
		return 0, fmt.Errorf("get collection must return slice")
	}

	return sliceData.Len(), nil
}

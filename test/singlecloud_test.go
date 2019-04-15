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

var gToken string

type Token struct {
	Token string `json:"token"`
}

type TestResource struct {
	CollectionUrl string                 `json:"collectionUrl"`
	ResourceUrl   string                 `json:"resourceUrl"`
	Params        map[string]interface{} `json:"params"`
}

type TestLogin struct {
	LoginUrl    string                 `json:"loginUrl"`
	LoginParams map[string]interface{} `json:"loginParams"`
}

func loadTestLogin(file string) (*TestLogin, error) {
	var login TestLogin
	if err := load(file, &login); err != nil {
		return nil, err
	} else {
		return &login, nil
	}
}

func loadTestResource(file string) (*TestResource, error) {
	var resource TestResource
	if err := load(file, &resource); err != nil {
		return nil, err
	} else {
		return &resource, nil
	}
}

func load(file string, resource interface{}) error {
	if data, err := ioutil.ReadFile(file); err != nil {
		return err
	} else {
		if err := json.Unmarshal(data, resource); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func runTestServer() {
	logger.InitLogger()

	server, err := server.NewServer()
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	if err := server.Run("0.0.0.0:1234"); err != nil {
		panic("server run failed:" + err.Error())
	}
}

func importTestCluster(loginResource *TestLogin, clusterResource *TestResource) error {
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

	var token Token
	err = sendRequest("POST", loginResource.LoginUrl, parseBodyFromParams(loginResource.LoginParams), &token)
	if err != nil {
		return fmt.Errorf("login failed:%s", err.Error())
	}
	gToken = token.Token

	clusterResource.Params["yaml_"] = string(data)
	return sendRequest("POST", clusterResource.CollectionUrl, parseBodyFromParams(clusterResource.Params), nil)
}

func TestRunSingleCloud(t *testing.T) {
	go runTestServer()
	time.Sleep(1 * time.Second)
}

func TestCluster(t *testing.T) {
	loginResource, err := loadTestLogin("adminlogin.json")
	ut.Equal(t, err, nil)
	clusterResource, err := loadTestResource("cluster.json")
	ut.Equal(t, err, nil)
	err = importTestCluster(loginResource, clusterResource)
	ut.Equal(t, err, nil)

	existClusterNum, err := getCollectionNum(clusterResource.CollectionUrl)
	ut.Equal(t, err, nil)
	ut.Equal(t, existClusterNum, 1)

	var cluster types.Cluster
	err = sendRequest("GET", clusterResource.ResourceUrl, nil, &cluster)
	ut.Equal(t, err, nil)
	ut.Equal(t, cluster.Name, "sc-test-cluster1")
}

func TestNamespace(t *testing.T) {
	namespaceResource, err := loadTestResource("namespace.json")
	ut.Equal(t, err, nil)
	var namespace types.Namespace
	err = testOperatorResource(namespaceResource, &namespace)
	ut.Equal(t, err, nil)
	ut.Equal(t, namespace.Name, "sc-test-namespace1")
}

func TestConfigMap(t *testing.T) {
	configmapResource, err := loadTestResource("configmap.json")
	ut.Equal(t, err, nil)
	var configMap types.ConfigMap
	err = testOperatorResource(configmapResource, &configMap)
	ut.Equal(t, err, nil)
	ut.Equal(t, configMap.Name, "sc-test-configmap1")
	ut.Equal(t, configMap.Configs[0].Name, "sc-test-cm-dataname1")
	ut.Equal(t, configMap.Configs[0].Data, "sc-test-cm-data1")
}

func TestSecret(t *testing.T) {
	secretResource, err := loadTestResource("secret.json")
	ut.Equal(t, err, nil)
	var secret types.Secret
	err = testOperatorResource(secretResource, &secret)
	ut.Equal(t, err, nil)
	ut.Equal(t, secret.Name, "sc-test-secret1")
	ut.Equal(t, secret.Data["sc-test-secret-dataname1"], "emRucw==")
}

func TestDeployment(t *testing.T) {
	deployResource, err := loadTestResource("deployment.json")
	ut.Equal(t, err, nil)
	var deploy types.Deployment
	err = testOperatorResource(deployResource, &deploy)
	ut.Equal(t, err, nil)
	ut.Equal(t, deploy.Name, "sc-test-deployment1")
	ut.Equal(t, deploy.Containers[0].Name, "sc-test-containter1")
	ut.Equal(t, deploy.Containers[0].ConfigName, "sc-test-configmap1")
	ut.Equal(t, deploy.Containers[0].SecretName, "sc-test-secret1")

	serviceResource, err := loadTestResource("service.json")
	ut.Equal(t, err, nil)
	var service types.Service
	err = sendRequest("GET", serviceResource.ResourceUrl, nil, &service)
	ut.Equal(t, err, nil)
	ut.Equal(t, service.Name, "sc-test-deployment1")
	ut.Equal(t, service.ServiceType, "clusterip")
	ut.Equal(t, service.ExposedPorts[0].Name, "sc-test-port1")
	ut.Equal(t, service.ExposedPorts[0].Port, 22222)
	ut.Equal(t, service.ExposedPorts[0].Protocol, "tcp")

	ingressResource, err := loadTestResource("ingress.json")
	ut.Equal(t, err, nil)
	var ingress types.Ingress
	err = sendRequest("GET", ingressResource.ResourceUrl, nil, &ingress)
	ut.Equal(t, err, nil)
	ut.Equal(t, ingress.Name, "sc-test-deployment1")
	ut.Equal(t, ingress.Rules[0].Host, "sc-test-ingress1")
	ut.Equal(t, ingress.Rules[0].Paths[0].Path, "/etc/scingress")
	ut.Equal(t, ingress.Rules[0].Paths[0].ServicePort, 22222)
	ut.Equal(t, ingress.Rules[0].Paths[0].ServiceName, "sc-test-deployment1")
}

func TestCronJob(t *testing.T) {
	cronjobResource, err := loadTestResource("cronjob.json")
	ut.Equal(t, err, nil)
	var cronjob types.CronJob
	err = testOperatorResource(cronjobResource, &cronjob)
	ut.Equal(t, err, nil)
	ut.Equal(t, cronjob.Name, "sc-test-cronjob1")
	ut.Equal(t, cronjob.Schedule, "*/1 * * * *")
	ut.Equal(t, cronjob.Containers[0].Name, "sc-test-cronjob-containter1")
}

func TestJob(t *testing.T) {
	jobResource, err := loadTestResource("job.json")
	ut.Equal(t, err, nil)
	var job types.Job
	err = testOperatorResource(jobResource, &job)
	ut.Equal(t, err, nil)
	ut.Equal(t, job.Name, "sc-test-job1")
	ut.Equal(t, job.Containers[0].Name, "sc-test-job-containter1")
}

func TestLimitRange(t *testing.T) {
	limitsResource, err := loadTestResource("limitrange.json")
	ut.Equal(t, err, nil)
	var limit types.LimitRange
	err = testOperatorResource(limitsResource, &limit)
	ut.Equal(t, err, nil)
	ut.Equal(t, limit.Name, "sc-test-limitrange1")
	ut.Equal(t, limit.Max["cpu"], "200m")
	ut.Equal(t, limit.Max["memory"], "200Ki")
	ut.Equal(t, limit.Min["cpu"], "100m")
	ut.Equal(t, limit.Min["memory"], "100Ki")
}

func TestResourceQuota(t *testing.T) {
	quotaResource, err := loadTestResource("resourcequota.json")
	ut.Equal(t, err, nil)
	var quota types.ResourceQuota
	err = testOperatorResource(quotaResource, &quota)
	ut.Equal(t, err, nil)
	ut.Equal(t, quota.Name, "sc-test-resourcequota1")
	ut.Equal(t, quota.Limits["limits.cpu"], "200m")
	ut.Equal(t, quota.Limits["limits.memory"], "200Ki")
	ut.Equal(t, quota.Limits["requests.cpu"], "200m")
	ut.Equal(t, quota.Limits["requests.memory"], "200Ki")
}

func TestUser(t *testing.T) {
	userResource, err := loadTestResource("user.json")
	ut.Equal(t, err, nil)
	var user types.User
	err = testOperatorResource(userResource, &user)
	ut.Equal(t, err, nil)
	ut.Equal(t, user.Name, "sc-test-user1")

	userLogin, err := loadTestLogin("userlogin.json")
	ut.Equal(t, err, nil)
	var token Token
	err = sendRequest("POST", userLogin.LoginUrl, parseBodyFromParams(userLogin.LoginParams), &token)
	ut.Equal(t, err, nil)
	ut.Equal(t, token.Token != "", true)
	adminToken := gToken
	gToken = token.Token
	var testUser types.User
	err = sendRequest("GET", userResource.ResourceUrl, nil, &testUser)
	ut.Equal(t, err, nil)
	ut.Equal(t, testUser.Name, "sc-test-user1")
	gToken = adminToken
}

func TestDeleteResource(t *testing.T) {
	deleteResourceFiles := []string{"deployment.json", "configmap.json", "secret.json", "job.json", "cronjob.json",
		"limitrange.json", "resourcequota.json", "user.json", "namespace.json"}

	for _, resourceFile := range deleteResourceFiles {
		testResource, err := loadTestResource(resourceFile)
		ut.Equal(t, err, nil)
		err = sendRequest("DELETE", testResource.ResourceUrl, nil, nil)
		ut.Equal(t, err, nil)
	}
}

func parseBodyFromParams(params map[string]interface{}) io.Reader {
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

func testOperatorResource(resource *TestResource, result interface{}) error {
	existResourcesNum, err := getCollectionNum(resource.CollectionUrl)
	if err != nil {
		return err
	}

	if err := sendRequest("POST", resource.CollectionUrl, parseBodyFromParams(resource.Params), nil); err != nil {
		return err
	}

	currentResourcesNum, err := getCollectionNum(resource.CollectionUrl)
	if err != nil {
		return err
	}

	if currentResourcesNum != existResourcesNum+1 {
		return fmt.Errorf("expect resource num %d not %d", existResourcesNum+1, currentResourcesNum)
	}

	return sendRequest("GET", resource.ResourceUrl, nil, result)
}

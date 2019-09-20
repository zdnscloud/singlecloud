package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	ut "github.com/zdnscloud/cement/unittest"
	zkecore "github.com/zdnscloud/zke/core"

	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	//"github.com/zdnscloud/singlecloud/pkg/globaldns"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/server"
	"github.com/zdnscloud/singlecloud/storage"
)

var (
	gToken       string
	gClusterName string
)

type Token struct {
	Token string `json:"token"`
}

type TestResource struct {
	CollectionUrl string                 `json:"collectionUrl"`
	ResourceUrl   string                 `json:"resourceUrl"`
	ImportUrl     string                 `json:"importUrl"`
	Params        map[string]interface{} `json:"params"`
}

type TestLogin struct {
	LoginUrl    string                 `json:"loginUrl"`
	LoginParams map[string]interface{} `json:"loginParams"`
}

type TestResourceCollection struct {
	Type         string            `json:"type"`
	ResourceType string            `json:"resourceType"`
	Links        map[string]string `json:"links"`
	Data         interface{}       `json:"data"`
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
	log.InitLogger(log.Debug)
	eventBus := pubsub.New(1000)

	db, err := storage.New("")
	if err != nil {
		panic("init db failed: " + err.Error())
	}

	/*
		if err := globaldns.New("0.0.0.0:8080", eventBus); err != nil {
			panic("create globaldns failed: " + err.Error())
		}
	*/

	authenticator, err := authentication.New("", db)
	if err != nil {
		panic("init authentication failed: " + err.Error())
	}

	authorizer, err := authorization.New(db)
	if err != nil {
		panic("init authorization failed: " + err.Error())
	}

	server, err := server.NewServer(authenticator.MiddlewareFunc())
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	if err := server.RegisterHandler(authenticator); err != nil {
		panic("register authorization handler failed:" + err.Error())
	}

	agent := clusteragent.New()
	app := handler.NewApp(authenticator, authorizer, eventBus, agent, db, "")
	if err := server.RegisterHandler(app); err != nil {
		panic("register resource handler failed:" + err.Error())
	}

	if err := server.Run("0.0.0.0:1234"); err != nil {
		panic("server run failed:" + err.Error())
	}
}

func getClusterName(stateJson []byte) (string, error) {
	state := &zkecore.FullState{}
	if err := json.Unmarshal(stateJson, state); err != nil {
		return "", err
	}
	return state.CurrentState.ZKEConfig.ClusterName, nil
}

func login(loginResource *TestLogin) error {
	var token Token
	if err := sendRequest("POST", loginResource.LoginUrl, parseBodyFromParams(loginResource.LoginParams), &token); err != nil {
		return fmt.Errorf("login failed:%s", err.Error())
	}
	gToken = token.Token
	return nil
}

func TestRunSingleCloud(t *testing.T) {
	go runTestServer()
	time.Sleep(1 * time.Second)
}

func TestCluster(t *testing.T) {
	loginResource, err := loadTestLogin("adminlogin.json")
	ut.Equal(t, err, nil)
	err = login(loginResource)
	ut.Equal(t, err, nil)

	clusterResource, err := loadTestResource("cluster.json")
	ut.Equal(t, err, nil)

	f, err := os.Open("cluster.zkestate")
	ut.Equal(t, err, nil)
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	ut.Equal(t, err, nil)

	clusterName, err := getClusterName(data)
	ut.Equal(t, err, nil)
	gClusterName = clusterName

	err = sendRequest("POST", fmt.Sprintf(clusterResource.ImportUrl, gClusterName), bytes.NewBuffer(data), nil)
	ut.Equal(t, err, nil)

	existClusterNum, err := getCollectionNum(clusterResource.CollectionUrl)
	ut.Equal(t, err, nil)
	ut.Equal(t, existClusterNum, 1)

	var cluster types.Cluster
	for cluster.Status != "Running" {
		err = sendRequest("GET", fmt.Sprintf(clusterResource.ResourceUrl, gClusterName), nil, &cluster)
		ut.Equal(t, err, nil)
		ut.Equal(t, cluster.Name, clusterName)
		time.Sleep(5 * time.Second)
	}
	ut.Equal(t, string(cluster.Status), "Running")
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
	ut.Equal(t, len(secret.Data), 1)
	ut.Equal(t, secret.Data[0].Key, "sc-test-secret-dataname1")
	ut.Equal(t, secret.Data[0].Value, "emRucw==")
}

func TestDeployment(t *testing.T) {
	deployResource, err := loadTestResource("deployment.json")
	ut.Equal(t, err, nil)
	var deploy types.Deployment
	err = testOperatorResource(deployResource, &deploy)
	ut.Equal(t, err, nil)
	ut.Equal(t, deploy.Name, "sc-test-deployment1")
	ut.Equal(t, deploy.Containers[0].Name, "sc-test-containter1")
	ut.Equal(t, deploy.Containers[0].Env[0].Name, "TESTENV1")
	ut.Equal(t, deploy.Containers[0].Env[0].Value, "testenv1")

	ut.Equal(t, len(deploy.Containers[0].Volumes), 3)
	for _, volume := range deploy.Containers[0].Volumes {
		switch volume.Type {
		case "configmap":
			ut.Equal(t, volume.Name, "sc-test-configmap1")
			ut.Equal(t, volume.MountPath, "/etc/scconfig")
		case "secret":
			ut.Equal(t, volume.Name, "sc-test-secret1")
			ut.Equal(t, volume.MountPath, "/etc/scsecret")
		case "persistentVolume":
			if volume.Name == "sc-test-emptydir1" {
				ut.Equal(t, volume.MountPath, "/etc/scdmtestpvc11")
			} else {
				ut.Equal(t, volume.Name, "sc-test-lvm-pvc1")
				ut.Equal(t, volume.MountPath, "/etc/scdmtestpvc12")
			}
		}

	}
}

func TestService(t *testing.T) {
	serviceResource, err := loadTestResource("service.json")
	ut.Equal(t, err, nil)
	var service types.Service
	err = testOperatorResource(serviceResource, &service)
	ut.Equal(t, err, nil)
	ut.Equal(t, service.Name, "sc-test-deployment1")
	ut.Equal(t, service.ServiceType, "clusterip")
	ut.Equal(t, service.ExposedPorts[0].Name, "sc-test-port1")
	ut.Equal(t, service.ExposedPorts[0].Port, 44444)
	ut.Equal(t, service.ExposedPorts[0].TargetPort, 22222)
	ut.Equal(t, service.ExposedPorts[0].Protocol, "tcp")
}

func TestIngress(t *testing.T) {
	ingressResource, err := loadTestResource("ingress.json")
	ut.Equal(t, err, nil)
	var ingress types.Ingress
	err = testOperatorResource(ingressResource, &ingress)
	ut.Equal(t, err, nil)
	ut.Equal(t, ingress.Name, "sc-test-ing1")
	ut.Equal(t, ingress.Rules[0].Host, "sc.test.ing")
	ut.Equal(t, ingress.Rules[0].Path, "/")
	ut.Equal(t, ingress.Rules[0].ServicePort, 44444)
	ut.Equal(t, ingress.Rules[0].ServiceName, "sc-test-deployment1")
}

func TestStatefulSet(t *testing.T) {
	statefulsetResource, err := loadTestResource("statefulset.json")
	ut.Equal(t, err, nil)
	var statefulset types.StatefulSet
	err = testOperatorResource(statefulsetResource, &statefulset)
	ut.Equal(t, err, nil)
	ut.Equal(t, statefulset.Name, "sc-test-statefulset1")
	ut.Equal(t, statefulset.Containers[0].Name, "sc-test-containter1")
	ut.Equal(t, statefulset.Containers[0].Env[0].Name, "TESTENV1")
	ut.Equal(t, statefulset.Containers[0].Env[0].Value, "testenv1")
	ut.Equal(t, len(statefulset.Containers[0].Volumes), 3)
	ut.Equal(t, len(statefulset.PersistentVolumes), 1)

	for _, volume := range statefulset.Containers[0].Volumes {
		switch volume.Type {
		case "configmap":
			ut.Equal(t, volume.Name, "sc-test-configmap1")
			ut.Equal(t, volume.MountPath, "/etc/scconfig")
		case "secret":
			ut.Equal(t, volume.Name, "sc-test-secret1")
			ut.Equal(t, volume.MountPath, "/etc/scsecret")
		case "persistentVolume":
			if volume.Name == "sc-test-emptydir1" {
				ut.Equal(t, volume.MountPath, "/etc/scststestpvc21")
			} else {
				ut.Equal(t, volume.Name, "sc-test-lvm-pvc2")
				ut.Equal(t, volume.MountPath, "/etc/scststestpvc22")
			}
		}
	}

	for _, template := range statefulset.PersistentVolumes {
		switch template.StorageClassName {
		case types.StorageClassNameTemp:
			ut.Equal(t, template.Name, "sc-test-emptydir1")
			ut.Equal(t, template.Size, "100Mi")
		case types.StorageClassNameLVM:
			ut.Equal(t, template.Name, "sc-test-lvm-pvc2")
			ut.Equal(t, template.Size, "200Mi")
		case types.StorageClassNameCeph:
		}
	}
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

func TestGetPod(t *testing.T) {
	deployPodResource, err := loadTestResource("deployment_pod.json")
	ut.Equal(t, err, nil)
	podNum, err := getCollectionNum(fmt.Sprintf(deployPodResource.CollectionUrl, gClusterName))
	ut.Equal(t, err, nil)
	ut.Equal(t, podNum, 2)
	stsPodResource, err := loadTestResource("statefulset_pod.json")
	ut.Equal(t, err, nil)
	podNum, err = getCollectionNum(fmt.Sprintf(stsPodResource.CollectionUrl, gClusterName))
	ut.Equal(t, err, nil)
	ut.Equal(t, podNum != 0, true)
}

func testGetStorageClass(t *testing.T) {
	scResource, err := loadTestResource("storageclass.json")
	ut.Equal(t, err, nil)
	var collection TestResourceCollection
	err = sendRequest("GET", fmt.Sprintf(scResource.CollectionUrl, gClusterName), nil, &collection)
	ut.Equal(t, err, nil)
	sliceData := reflect.ValueOf(collection.Data)
	ut.Equal(t, sliceData.Kind(), reflect.Slice)
	ut.Equal(t, sliceData.Len(), 2)
	existLVM := false
	existCephNFS := false
	for i := 0; i < sliceData.Len(); i++ {
		c := sliceData.Index(i).Interface()
		object, ok := c.(map[string]interface{})
		ut.Equal(t, ok, true)
		switch object["name"] {
		case types.StorageClassNameLVM:
			existLVM = true
		case types.StorageClassNameCeph:
			existCephNFS = true
		}
	}

	ut.Equal(t, existLVM || existCephNFS, true)
}

func TestDeleteResource(t *testing.T) {
	defer os.Remove("singlecloud.db")
	deleteResourceFiles := []string{"deployment.json", "statefulset.json", "configmap.json", "secret.json", "job.json", "cronjob.json", "limitrange.json", "resourcequota.json", "user.json", "namespace.json", "cluster.json"}

	for _, resourceFile := range deleteResourceFiles {
		testResource, err := loadTestResource(resourceFile)
		ut.Equal(t, err, nil)
		resourceUrl := testResource.ResourceUrl
		if strings.Contains(testResource.ResourceUrl, "/clusters/%s") {
			resourceUrl = fmt.Sprintf(testResource.ResourceUrl, gClusterName)
		}

		err = sendRequest("DELETE", resourceUrl, nil, nil)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		if result != nil {
			return json.Unmarshal(body, result)
		} else {
			return nil
		}
	case http.StatusCreated:
		fallthrough
	case http.StatusNoContent:
		return nil
	default:
		errInfo := struct {
			Message string `json:"message"`
		}{}
		json.Unmarshal(body, &errInfo)
		return fmt.Errorf("%s %s failed with status %v : %s", method, url, resp.StatusCode, errInfo.Message)
	}
}

func getCollectionNum(url string) (int, error) {
	var oldcollection TestResourceCollection
	err := sendRequest("GET", url, nil, &oldcollection)
	if err != nil {
		return 0, fmt.Errorf("get old collection failed: %s", err.Error())
	}

	if oldcollection.Data == nil {
		return 0, nil
	}

	sliceData := reflect.ValueOf(oldcollection.Data)
	if sliceData.Kind() == reflect.Slice {
		return sliceData.Len(), nil
	}

	return 0, fmt.Errorf("get collection must return slice")
}

func testOperatorResource(resource *TestResource, result interface{}) error {
	collectionUrl := resource.CollectionUrl
	if strings.Contains(resource.CollectionUrl, "/clusters/%s") {
		collectionUrl = fmt.Sprintf(resource.CollectionUrl, gClusterName)
	}

	existResourcesNum, err := getCollectionNum(collectionUrl)
	if err != nil {
		return err
	}

	if err := sendRequest("POST", collectionUrl, parseBodyFromParams(resource.Params), nil); err != nil {
		return err
	}

	currentResourcesNum, err := getCollectionNum(collectionUrl)
	if err != nil {
		return err
	}

	if currentResourcesNum != existResourcesNum+1 {
		return fmt.Errorf("expect resource num %d not %d", existResourcesNum+1, currentResourcesNum)
	}

	resourceUrl := resource.ResourceUrl
	if strings.Contains(resource.ResourceUrl, "/clusters/%s") {
		resourceUrl = fmt.Sprintf(resource.ResourceUrl, gClusterName)
	}
	return sendRequest("GET", resourceUrl, nil, result)
}

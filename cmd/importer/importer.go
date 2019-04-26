package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/kyokomi/emoji"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 48, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	reset   = string([]byte{27, 91, 48, 109})
)

func getDefaultConfigPath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("get current user failed:%s", err.Error())
	}
	return filepath.Join(usr.HomeDir, ".kube", "config")
}

func login(addr string, user, password string) (string, error) {
	client := &http.Client{}
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/users/%s?action=login", addr, user)
	requestBody, _ := json.Marshal(map[string]string{
		"password": hashPassword(password),
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	token := struct {
		Token string `json:"token"`
	}{}
	json.Unmarshal(body, &token)
	return token.Token, nil
}

func hashPassword(password string) string {
	pwHash := sha1.Sum([]byte(password))
	return hex.EncodeToString(pwHash[:])
}

func createCluster(addr, token, clusterName string, data []byte) error {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters", addr)
	requestBody, _ := json.Marshal(map[string]string{
		"name":  clusterName,
		"yaml_": string(data),
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("send request failed:%s", err.Error())
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 201 {
		return nil
	}

	errInfo := struct {
		Message string `json:"message"`
	}{}
	json.Unmarshal(body, &errInfo)
	return errors.New(errInfo.Message)
}

func deleteZcloudProxyDeployment(addr, token, clusterName string) error {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/zcloud/deployments/zcloud-proxy", addr, clusterName)
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("send request failed:%s", err.Error())
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 204 {
		log.Println("delete exist zcloud-proxy deployment")
		return nil
	}

	if resp.StatusCode == 422 {
		log.Println("not exist zcloud-proxy deployment, this is a new cluster")
		return nil
	}
	errInfo := struct {
		Message string `json:"message"`
	}{}
	json.Unmarshal(body, &errInfo)
	return errors.New(errInfo.Message)
}

func createZcloudProxyDeployment(addr, token, clusterName string) error {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s/namespaces/zcloud/deployments", addr, clusterName)
	deployment := types.Deployment{
		Name:     "zcloud-proxy",
		Replicas: 1,
		Containers: []types.Container{
			types.Container{
				Name:    "zcloud-proxy",
				Image:   "zdnscloud/zcloud-proxy:v1.0.0",
				Command: []string{"agent"},
				Args:    []string{"-server", addr, "-agentKey", clusterName},
			},
		},
	}
	requestBody, _ := json.Marshal(deployment)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("send request failed:%s", err.Error())
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 201 {
		return nil
	}

	errInfo := struct {
		Message string `json:"message"`
	}{}
	json.Unmarshal(body, &errInfo)
	return errors.New(errInfo.Message)
}

func main() {
	var addr, k8sconfig, clusterName, adminPassword string
	flag.StringVar(&addr, "server", "127.0.0.1:80", "singlecloud server listen address")
	flag.StringVar(&k8sconfig, "k8sconfig", getDefaultConfigPath(), "k8s config file path")
	flag.StringVar(&clusterName, "name", "local", "import cluster with which name")
	flag.StringVar(&adminPassword, "passwd", "zdns", "admin password for singlecloud")
	flag.Parse()

	f, err := os.Open(k8sconfig)
	if err != nil {
		log.Fatalf("open %s failed:%s", k8sconfig, err.Error())
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("read %s failed:%s", k8sconfig, err.Error())
	}

	token, err := login(addr, "admin", adminPassword)
	if err != nil {
		log.Fatalf("get token failed:%s", err.Error())
	}

	err = createCluster(addr, token, clusterName, data)
	if err != nil {
		log.Fatalf("create cluster failed:%s", err.Error())
	}

	err = deleteZcloudProxyDeployment(addr, token, clusterName)
	if err != nil {
		log.Fatalf("delete zcloud-proxy deployment failed:%s", err.Error())
	}

	err = createZcloudProxyDeployment(addr, token, clusterName)
	if err == nil {
		fmt.Printf("%s|%s %s %s\n", emoji.Sprint(":+1:"), green, "import succeed", reset)
	} else {
		log.Fatalf("create zcloud-proxy deployment failed:%s", err.Error())
	}
}

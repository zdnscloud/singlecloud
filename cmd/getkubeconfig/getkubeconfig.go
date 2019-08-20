package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

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

func getClusterKubeConfig(addr, token, clusterName string) string {
	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters/%s?action=getkubeconfig", addr, clusterName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		log.Fatalf("struct getkubeconfig action http request failed %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("send request failed:%s", err.Error())
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("read reposence body failed %s", err.Error())
	}

	kubeConfig := struct {
		Name   string `json:"name"`
		Config string `json:"config"`
	}{}
	if err := json.Unmarshal(body, &kubeConfig); err != nil {
		log.Fatalf("unmarshal kubeConfig failed %s", err.Error())
	}
	if kubeConfig.Config == "" {
		log.Fatalf("got empty kubeConfig from singlecloud")
	}
	return kubeConfig.Config
}

func main() {
	var addr, clusterName, adminPassword string
	flag.StringVar(&addr, "server", "127.0.0.1:80", "singlecloud server listen address")
	flag.StringVar(&adminPassword, "passwd", "zdns", "admin password for singlecloud")
	flag.StringVar(&clusterName, "cluster", "local", "cluster name")
	flag.Parse()

	token, err := login(addr, "admin", adminPassword)
	if err != nil {
		log.Fatalf("get token failed %s", err.Error())
	}

	kubeConfig := getClusterKubeConfig(addr, token, clusterName)
	fileName := "kube_config_" + clusterName + ".yml"

	if err := ioutil.WriteFile(fileName, []byte(kubeConfig), 0644); err != nil {
		log.Fatalf("write kubeconfig file failed %s", err)
	}
	log.Printf("success get kubeconfig %s", fileName)
}

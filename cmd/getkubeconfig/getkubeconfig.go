package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func login(addr string, user, password string) (string, error) {
	url := fmt.Sprintf("https://%s/apis/zcloud.cn/v1/users/%s?action=login", addr, user)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		errInfo := struct {
			Message string `json:"message"`
		}{}
		json.Unmarshal(body, &errInfo)
		return "", errors.New(errInfo.Message)
	}

	token := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(body, &token); err != nil {
		return "", err
	}
	return token.Token, nil
}

func hashPassword(password string) string {
	pwHash := sha1.Sum([]byte(password))
	return hex.EncodeToString(pwHash[:])
}

func getClusterKubeConfig(addr, token, clusterName string) (string, error) {
	url := fmt.Sprintf("https://%s/apis/zcloud.cn/v1/clusters/%s/kubeconfigs/kube-admin", addr, clusterName)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		errInfo := struct {
			Message string `json:"message"`
		}{}
		json.Unmarshal(body, &errInfo)
		return "", errors.New(errInfo.Message)
	}

	kubeConfig := struct {
		User   string `json:"user"`
		Config string `json:"kubeConfig"`
	}{}
	if err := json.Unmarshal(body, &kubeConfig); err != nil {
		return "", err
	}
	return kubeConfig.Config, nil
}

func main() {
	var addr, clusterName, adminPassword string
	flag.StringVar(&addr, "server", "127.0.0.1:443", "singlecloud server listen address")
	flag.StringVar(&adminPassword, "passwd", "zcloud", "admin password for singlecloud")
	flag.StringVar(&clusterName, "cluster", "local", "cluster name")
	flag.Parse()

	token, err := login(addr, "admin", adminPassword)
	if err != nil {
		log.Fatalf("get token failed %s", err.Error())
	}

	kubeConfig, err := getClusterKubeConfig(addr, token, clusterName)
	if err != nil {
		log.Fatalf("get kubeConfig from singlecloud failed %s", err.Error())
	}

	fileName := "kube_config_" + clusterName + ".yml"
	if err := ioutil.WriteFile(fileName, []byte(kubeConfig), 0644); err != nil {
		log.Fatalf("write kubeconfig file failed %s", err)
	}
	log.Printf("success get kubeconfig %s", fileName)
}

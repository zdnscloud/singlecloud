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
	"os"
	"os/user"
	"path/filepath"

	"github.com/kyokomi/emoji"
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

	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/users/admin?action=login", addr)
	pwHash := sha1.Sum([]byte(adminPassword))
	requestBody, _ := json.Marshal(map[string]string{
		"user":     "admin",
		"password": hex.EncodeToString(pwHash[:]),
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("get token failed:%s", err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	token := struct {
		Token string `json:"token"`
	}{}
	json.Unmarshal(body, &token)

	url = fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters", addr)
	requestBody, _ = json.Marshal(map[string]string{
		"name":  clusterName,
		"yaml_": string(data),
	})
	req, _ = http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token.Token)
	resp, err = client.Do(req)
	if err != nil {
		log.Fatalf("send request failed:%s", err.Error())
	}

	defer resp.Body.Close()
	body, _ = ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 201 {
		fmt.Printf("%s|%s %s %s\n", emoji.Sprint(":+1:"), green, "import succeed", reset)
	} else {
		errInfo := struct {
			Message string `json:"message"`
		}{}
		json.Unmarshal(body, &errInfo)
		fmt.Printf("%s|%s %s: %s %s\n", emoji.Sprint(":broken_heart:"), red, "import failed", errInfo.Message, reset)
	}
}

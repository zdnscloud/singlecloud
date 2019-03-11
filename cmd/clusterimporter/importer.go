package main

import (
	"bytes"
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

func getDefaultConfigPath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("get current user failed:%s", err.Error())
	}
	return filepath.Join(usr.HomeDir, ".kube", "config")
}

func main() {
	var addr, k8sconfig string
	flag.StringVar(&addr, "server", "127.0.0.1:80", "singlecloud server listen address")
	flag.StringVar(&k8sconfig, "k8sconfig", getDefaultConfigPath(), "k8s config file path")
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

	url := fmt.Sprintf("http://%s/apis/zcloud.cn/v1/clusters", addr)
	requestBody, _ := json.Marshal(map[string]string{
		"name":  "local",
		"yaml_": string(data),
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("send request failed:%s", err.Error())
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 201 {
		output := emoji.Sprint("import succeed :+1:")
		fmt.Printf("%s\n", output)
	} else {
		output := emoji.Sprint("import failed :broken_heart:")
		fmt.Printf("%s,%s\n", output, string(body))
	}
}

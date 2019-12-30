package alarm

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	Subject                               = "Singlecloud Alarm"
	ClusterThresholdConfigmapName         = "resource-threshold"
	NamespaceThresholdConfigmapNamePrefix = "resource-threshold-"
	ThresholdConfigmapNamespace           = "zcloud"
)

var ClusterKinds = []string{"Node", "node", "Cluster", "cluster"}

func SendMail(cli client.Client, alarm *types.Alarm) {
	mailConn, err := getSender(cli)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		log.Warnf("get mail sender failed. %s", err)
	}
	mailTo, err := getRecipient(cli, alarm)
	if err != nil {
		log.Warnf("get mail recipient failed. %s", err)
	}

	port, _ := strconv.Atoi(mailConn["port"])
	m := gomail.NewMessage()
	m.SetHeader("From", mailConn["user"])
	m.SetHeader("To", mailTo...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", genMessage(alarm))

	d := gomail.NewDialer(mailConn["host"], port, mailConn["user"], mailConn["pass"])
	if err := d.DialAndSend(m); err != nil {
		log.Warnf("send mail to %s failed:%s", mailTo, err.Error())
	}
	return
}

func getSender(cli client.Client) (map[string]string, error) {
	sender := make(map[string]string)
	cm, err := getConfigMap(cli, ClusterThresholdConfigmapName)
	if err != nil {
		return nil, err
	}
	data, ok := cm.Data["mailFrom"]
	if ok {
		var mail types.Mail
		if err := json.Unmarshal([]byte(data), &mail); err != nil {
			return nil, err
		}
		sender["user"] = mail.User
		sender["pass"] = mail.Password
		sender["host"] = mail.Host
		sender["port"] = mail.Port
	}
	return sender, nil
}

func getRecipient(cli client.Client, alarm *types.Alarm) ([]string, error) {
	var recipient []string
	var configMaps []corev1.ConfigMap
	clusterCm, err := getConfigMap(cli, ClusterThresholdConfigmapName)
	if err != nil {
		return nil, err
	}
	configMaps = append(configMaps, clusterCm)
	if slice.SliceIndex(ClusterKinds, alarm.Kind) == -1 && len(alarm.Namespace) != 0 {
		namespaceCm, err := getConfigMap(cli, NamespaceThresholdConfigmapNamePrefix+alarm.Namespace)
		if err == nil {
			configMaps = append(configMaps, namespaceCm)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
	}
	for _, cm := range configMaps {
		for _, mail := range genMailTo(cm) {
			recipient = append(recipient, mail)
		}
	}
	return recipient, nil
}

func genMailTo(cm corev1.ConfigMap) []string {
	data, ok := cm.Data["mailTo"]
	if ok {
		data := strings.Trim(data, "[]\"")
		return strings.Split(data, "\",\"")
	} else {
		return nil
	}
}

func genMessage(alarm *types.Alarm) string {
	info, _ := json.Marshal(alarm)
	return string(info)
}

func getConfigMap(cli client.Client, name string) (corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{ThresholdConfigmapNamespace, name}, &cm); err != nil {
		return cm, err
	}
	return cm, nil
}

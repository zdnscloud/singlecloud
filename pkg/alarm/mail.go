package alarm

import (
	"context"
	"encoding/json"
	"gopkg.in/gomail.v2"
	"strconv"
	"strings"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/pkg/types"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	Subject                               = "Singlecloud Alarm"
	ClusterThresholdConfigmapName         = "resource-threshold"
	NamespaceThresholdConfigmapNamePrefix = "resource-threshold-"
	ThresholdConfigmapNamespace           = "zcloud"
)

var ClusterKinds = []string{"Node", "node", "Cluster", "cluster"}

func SendMail(cli client.Client, alarm *Alarm) {
	mailConn := getSender(cli)
	mailTo := getRecipient(cli, alarm)
	if len(mailConn) == 0 || len(mailTo) == 0 {
		log.Warnf("Mail contacts are empty, return")
		return
	}

	port, _ := strconv.Atoi(mailConn["port"])
	m := gomail.NewMessage()
	m.SetHeader("From", mailConn["user"])
	m.SetHeader("To", mailTo...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", genMessage(alarm))

	d := gomail.NewDialer(mailConn["host"], port, mailConn["user"], mailConn["pass"])
	if err := d.DialAndSend(m); err != nil {
		log.Warnf("Send mail to %s failed:%s", mailTo, err.Error())
	} else {
		log.Infof("Send mail to %s success", mailTo)
	}
	return
}

func getSender(cli client.Client) map[string]string {
	sender := make(map[string]string)
	cm := corev1.ConfigMap{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{ThresholdConfigmapNamespace, ClusterThresholdConfigmapName}, &cm); err != nil {
		log.Warnf("Get configmap %s failed:%s", ClusterThresholdConfigmapName, err.Error())
		return sender
	}
	data, ok := cm.Data["mailFrom"]
	if ok {
		var mail types.Mail
		json.Unmarshal([]byte(data), &mail)
		sender["user"] = mail.User
		sender["pass"] = mail.Password
		sender["host"] = mail.Host
		sender["port"] = mail.Port
	}
	return sender
}

func getRecipient(cli client.Client, alarm *Alarm) []string {
	var recipient, configMapsName []string
	configMapsName = append(configMapsName, ClusterThresholdConfigmapName)
	if slice.SliceIndex(ClusterKinds, alarm.Kind) == -1 && len(alarm.Namespace) != 0 {
		name := NamespaceThresholdConfigmapNamePrefix + alarm.Namespace
		configMapsName = append(configMapsName, name)
	}
	for _, name := range configMapsName {
		cm := corev1.ConfigMap{}
		if err := cli.Get(context.TODO(), k8stypes.NamespacedName{ThresholdConfigmapNamespace, name}, &cm); err != nil {
			log.Warnf("Get configmap %s failed:%s", name, err.Error())
			return recipient
		}
		for _, mail := range genMailTo(cm) {
			recipient = append(recipient, mail)
		}
	}
	return recipient
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

func genMessage(alarm *Alarm) string {
	info, _ := json.Marshal(*alarm)
	return string(info)
}

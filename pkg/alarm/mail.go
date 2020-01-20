package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/gomail.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	Subject                     = "Singlecloud Alarm"
	ThresholdConfigmapName      = "resource-threshold"
	ThresholdConfigmapNamespace = "zcloud"
)

var ClusterKinds = []string{"Node", "node", "Cluster", "cluster"}

func SendMail(cli client.Client, alarm *types.Alarm) {
	mailConn, mailTo, err := getAdminMail(cli)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		log.Warnf("get mail sender failed. %s", err)
		return
	}

	port, _ := strconv.Atoi(mailConn.Port)
	m := gomail.NewMessage()
	m.SetHeader("From", mailConn.User)
	m.SetHeader("To", mailTo...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", genMessage(alarm))

	d := gomail.NewDialer(mailConn.Host, port, mailConn.User, mailConn.Password)
	if err := d.DialAndSend(m); err != nil {
		log.Warnf("send mail to %s failed:%s", mailTo, err.Error())
	}
	return
}

func getAdminMail(cli client.Client) (types.Mail, []string, error) {
	var mailFrom types.Mail
	var mailTo []string
	cm, err := getConfigMap(cli, ThresholdConfigmapName)
	if err != nil {
		return mailFrom, mailTo, err
	}
	if data, ok := cm.Data["mailFrom"]; ok {
		if err := json.Unmarshal([]byte(data), &mailFrom); err != nil {
			return mailFrom, mailTo, err
		}
	} else {
		return mailFrom, mailTo, fmt.Errorf("can not found mailFrom in configmap %s", ThresholdConfigmapName)
	}
	if data, ok := cm.Data["mailTo"]; ok {
		if err := json.Unmarshal([]byte(data), &mailTo); err != nil {
			return mailFrom, mailTo, err
		}
	} else {
		return mailFrom, mailTo, fmt.Errorf("can not found mailTo in configmap %s", ThresholdConfigmapName)
	}
	return mailFrom, mailTo, nil
}

func genMessage(alarm *types.Alarm) string {
	info, _ := json.Marshal(alarm)
	return string(info)
}

func getConfigMap(cli client.Client, name string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{ThresholdConfigmapNamespace, name}, cm)
	return cm, err
}

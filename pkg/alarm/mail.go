package alarm

import (
	"encoding/json"
	"fmt"

	"gopkg.in/gomail.v2"

	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	Subject = "Singlecloud Alarm"
)

var ClusterKinds = []string{"Node", "node", "Cluster", "cluster"}

func SendMail(alarm *types.Alarm, table kvzoo.Table) error {
	threshold, err := getThresholdFromDB(table, ThresholdConfigmapName)
	if err != nil {
		return fmt.Errorf("get threshold %s failed: %s", err.Error())
	}
	if len(threshold.MailFrom.User) == 0 ||
		len(threshold.MailFrom.Host) == 0 ||
		threshold.MailFrom.Port == 0 ||
		len(threshold.MailFrom.Password) == 0 ||
		len(threshold.MailTo) == 0 {
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", threshold.MailFrom.User)
	m.SetHeader("To", threshold.MailTo...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", genMessage(alarm))

	d := gomail.NewDialer(threshold.MailFrom.Host, threshold.MailFrom.Port, threshold.MailFrom.User, threshold.MailFrom.Password)
	return d.DialAndSend(m)
}

func getThresholdFromDB(table kvzoo.Table, name string) (*types.Threshold, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()
	value, err := tx.Get(name)
	if err != nil {
		return nil, err
	}
	var threshold types.Threshold
	if err := json.Unmarshal(value, &threshold); err != nil {
		return nil, err
	}
	return &threshold, nil
}

func genMessage(alarm *types.Alarm) string {
	info, _ := json.Marshal(alarm)
	return string(info)
}

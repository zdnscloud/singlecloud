package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	EventType  AlarmType = "Event"
	ZcloudType AlarmType = "Alarm"
)

type AlarmType string

type Alarm struct {
	resource.ResourceBase `json:",inline"`
	UID                   uint64           `json:"uid"`
	Time                  resource.ISOTime `json:"time" rest:"description=readonly"`
	Cluster               string           `json:"cluster" rest:"description=readonly"`
	Type                  AlarmType        `json:"type" rest:"description=readonly"`
	Namespace             string           `json:"namespace" rest:"description=readonly"`
	Kind                  string           `json:"kind" rest:"description=readonly"`
	Name                  string           `json:"name" rest:"description=readonly"`
	Reason                string           `json:"reason" rest:"description=readonly"`
	Message               string           `json:"message" rest:"description=readonly"`
	Acknowledged          bool             `json:"acknowledged"`
}

type Alarms []*Alarm

func (s Alarms) Len() int           { return len(s) }
func (s Alarms) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Alarms) Less(i, j int) bool { return s[i].UID > s[j].UID }

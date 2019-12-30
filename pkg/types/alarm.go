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
	UID                   uint64           `json:"-"`
	Time                  resource.ISOTime `json:"time,omitempty" rest:"description=readonly"`
	Cluster               string           `json:"cluster,omitempty" rest:"description=readonly"`
	Type                  AlarmType        `json:"type,omitempty" rest:"description=readonly"`
	Namespace             string           `json:"namespace,omitempty" rest:"description=readonly"`
	Kind                  string           `json:"kind,omitempty" rest:"description=readonly"`
	Name                  string           `json:"name,omitempty" rest:"description=readonly"`
	Reason                string           `json:"reason,omitempty" rest:"description=readonly"`
	Message               string           `json:"message,omitempty" rest:"description=readonly"`
	Acknowledged          bool             `json:"acknowledged"`
}

type Alarms []*Alarm

func (s Alarms) Len() int           { return len(s) }
func (s Alarms) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Alarms) Less(i, j int) bool { return s[i].UID < s[j].UID }

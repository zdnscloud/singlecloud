package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type FluentBitConfig struct {
	resource.ResourceBase `json:",inline"`
	Name                  string       `json:"-"`
	Workload              WorkloadConf `json:"workload" rest:"required=true"`
	RegExp                string       `json:"regexp" rest:"required=true"`
	Time_Key              string       `json:"timeKey,omitempty"`
	Time_Format           string       `json:"timeFormat,omitempty"`
}

type WorkloadConf struct {
	Namespace string `json:"namespace" rest:"required=true"`
	Name      string `json:"name" rest:"required=true"`
	Kind      string `json:"kind" rest:"required=true"`
}

type InstanceConf struct {
	Include string
	Input   InputConf
	Filter  FilterConf
	Output  OutputConf
	Parser  ParserConf
}

type InputConf struct {
	Tag  string
	Path string
	DB   string
}

type FilterConf struct {
	Match  string
	Parser string
}

type OutputConf struct {
	Match           string
	Logstash_Prefix string
}

type ParserConf struct {
	Name        string
	Regex       string
	Time_Key    string
	Time_Format string
}

func (e FluentBitConfig) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

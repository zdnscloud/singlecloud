package config

import (
	"errors"

	"github.com/zdnscloud/cement/configure"
	"github.com/zdnscloud/cement/log"
)

type SinglecloudConf struct {
	Path   string     `yaml:"-"`
	Server ServerConf `yaml:"server"`
	DB     DBConf     `yaml:"db"`
	Chart  ChartConf  `yaml:"chart"`
}

type ServerConf struct {
	Addr    string `yaml:"addr"`
	DNSAddr string `yaml:"dns_addr"`
	CasAddr string `yaml:"cas_addr"`
}

type DBConf struct {
	Path        string `yaml:"path"`
	Port        int    `yaml:"port"`
	Role        string `yaml:"role"`
	SlaveDBAddr string `yaml:"slave_db_addr"`
}

type ChartConf struct {
	Path string `yaml:"path"`
	Repo string `yaml:"repo"`
}

func CreateDefaultConfig() SinglecloudConf {
	return SinglecloudConf{
		Server: ServerConf{
			Addr: ":80",
		},
		DB: DBConf{
			Port: 6666,
			Role: "master",
		},
	}
}

func LoadConfig(path string) (*SinglecloudConf, error) {
	var conf SinglecloudConf
	conf.Path = path
	if err := conf.Reload(); err != nil {
		return nil, err
	}
	if err := conf.Verify(); err != nil {
		return nil, err
	}
	return &conf, nil
}

func (c *SinglecloudConf) Reload() error {
	newConf := CreateDefaultConfig()
	if err := configure.Load(&newConf, c.Path); err != nil {
		return err
	}
	newConf.Path = c.Path
	*c = newConf

	return nil
}

func (c *SinglecloudConf) Verify() error {
	if c.DB.Role != "master" && c.DB.Role != "slave" {
		return errors.New("db role can only as master or slave")
	}

	if c.DB.Role == "slave" && c.DB.SlaveDBAddr != "" {
		return errors.New("slave node cann't have other slaves")
	}

	if c.DB.Role == "master" && c.DB.SlaveDBAddr == "" {
		log.Warnf("no slave node is specified, if master node is crashed, data will be lost\n")
	}
	return nil
}

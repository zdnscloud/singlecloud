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
	Addr    string   `yaml:"addr"`
	DNSAddr string   `yaml:"dns_addr"`
	CasAddr string   `yaml:"cas_addr"`
	Role    RoleConf `yaml:"role"`
}

type RoleConf struct {
	Master bool `yaml:"master,omitempty"`
	Slave  bool `yaml:"slave,omitempty"`
}

type DBConf struct {
	Path        string `yaml:"path"`
	Port        int    `yaml:"port"`
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
	if !c.Server.Role.Master && !c.Server.Role.Slave {
		c.Server.Role.Master = true
	}

	if c.Server.Role.Master && c.Server.Role.Slave {
		return errors.New("singlecloud can only run as master or as slave")
	}

	if c.Server.Role.Slave && c.DB.SlaveDBAddr != "" {
		return errors.New("slave node cann't have other slaves")
	}

	if c.Server.Role.Master && c.DB.SlaveDBAddr == "" {
		log.Warnf("no slave node is specified, if master node is crashed, data will be lost\n")
	}
	return nil
}

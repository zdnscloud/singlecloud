package config

import (
	"errors"

	"github.com/zdnscloud/cement/configure"
	"github.com/zdnscloud/cement/log"
)

type DBRole string

const (
	Master DBRole = "master"
	Slave  DBRole = "slave"
)

type SinglecloudConf struct {
	Path     string         `yaml:"-"`
	Server   ServerConf     `yaml:"server"`
	DB       DBConf         `yaml:"db"`
	Chart    ChartConf      `yaml:"chart"`
	Registry RegistryCAConf `yaml:"registry"`
}

type ServerConf struct {
	Addr    string `yaml:"addr"`
	DNSAddr string `yaml:"dns_addr"`
	CasAddr string `yaml:"cas_addr"`
}

type DBConf struct {
	Path        string `yaml:"path"`
	Port        int    `yaml:"port"`
	Role        DBRole `yaml:"role"`
	SlaveDBAddr string `yaml:"slave_db_addr"`
	Version     string `yaml:"version"`
}

type ChartConf struct {
	Path string `yaml:"path"`
	Repo string `yaml:"repo"`
}

type RegistryCAConf struct {
	CaCertPath string `yaml:"ca_cert_path"`
	CaKeyPath  string `yaml:"ca_key_path"`
}

func CreateDefaultConfig() SinglecloudConf {
	return SinglecloudConf{
		Server: ServerConf{
			Addr: ":80",
		},
		DB: DBConf{
			Port: 6666,
			Role: Master,
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
	if c.DB.Role != Master && c.DB.Role != Slave {
		return errors.New("db role can only as master or slave")
	}

	if c.DB.Role == Slave && c.DB.SlaveDBAddr != "" {
		return errors.New("slave node cann't have other slaves")
	}

	if c.DB.Role == Master && c.DB.SlaveDBAddr == "" {
		log.Warnf("no slave node is specified, if master node is crashed, data will be lost\n")
	}

	if c.Registry.CaCertPath == "" || c.Registry.CaKeyPath == "" {
		return errors.New("registry ca must be specified")
	}
	return nil
}

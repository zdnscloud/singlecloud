package config

import (
	"github.com/zdnscloud/cement/configure"
)

type SingleCloudConf struct {
	Server ServerConf `yaml:"server"`
	Logger LogConf    `yaml:"logger"`
}

type ServerConf struct {
	Addr    string `yaml:"addr"`
	AuthDir string `yaml:"authDir"`
}

type LogConf struct {
	LogFile  string `yaml:"log_file"`
	FileSize int    `yaml:"size_in_byte"`
	Versions int    `yaml:"numbers_of_files"`
	Level    string `yaml:"level"`
}

func LoadConfig(filePath string) (*SingleCloudConf, error) {
	conf := &SingleCloudConf{}
	if err := configure.Load(conf, filePath); err != nil {
		return nil, err
	}

	return conf, nil
}

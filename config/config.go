package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

func Read(path string) (*Config, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := new(Config)
	err = yaml.Unmarshal(yamlFile, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

package main

import (
	"os"
	"resource-manager/api"
	"resource-manager/config"
)

const (
	configPathDefault = "config.yaml"
)

func main() {
	// получение пути конфигурационного файла
	configPath := os.Getenv("APP_CONFIG_PATH")
	if configPath == "" {
		configPath = configPathDefault
	}

	// чтение конфигурации
	cfg, err := config.Read(configPath)
	if err != nil {
		panic(err)
	}

	// передача конфигурации и запуск restAPI
	err = api.Run(cfg)
	if err != nil {
		panic(err)
	}
}

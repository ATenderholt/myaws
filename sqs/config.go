package sqs

import (
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"myaws/config"
	"myaws/docker"
	"myaws/log"
	"myaws/utils"
	"os"
	"path/filepath"
)

const Image = "softwaremill/elasticmq:1.3.4"

var basePath = filepath.Join(config.GetDataPath(), "sqs")
var configPath = filepath.Join(config.GetDataPath(), "sqs.conf")

var Container = docker.Container{
	Name:  "sqs",
	Image: Image,
	Mounts: []mount.Mount{
		{
			Source: basePath,
			Target: "/data",
			Type:   mount.TypeBind,
		},
		{
			Source: configPath,
			Target: "/opt/elasticmq.conf",
			Type:   mount.TypeBind,
		},
	},
	Ports: map[int]int{
		9324: 9324,
		9325: 9325,
	},
}

const configFileTemplate = `include classpath("application.conf")

messages-storage {
  enabled = true
}

aws {
  region = %s
  accountId = %s
}
`

func init() {
	err := utils.CreateDirs(basePath)
	if err != nil {
		msg := log.Error("Unable to create directory %s: %v", basePath, err)
		panic(msg)
	}

	writeConfigFile()
}

func writeConfigFile() {
	stat, err := os.Stat(configPath)
	if err == nil && stat.IsDir() {
		msg := log.Error("Expecting %s to be a file, but is a directory: %v", configPath, err)
		panic(msg)
	}

	if err == nil {
		log.Info("The file %s already exists, so returning without creating.")
		return
	}

	f, err := os.Create(configPath)
	if err != nil {
		msg := log.Error("Unable to open %s: %v", configPath, err)
		panic(msg)
	}

	contents := fmt.Sprintf(configFileTemplate, config.Region(), config.AccountNumber())
	_, err = f.WriteString(contents)
	if err != nil {
		msg := log.Error("Unable to write contents to %s: %v", configPath, err)
		panic(msg)
	}
}

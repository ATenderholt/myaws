package sqs

import (
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"myaws/docker"
	"myaws/log"
	"myaws/settings"
	"myaws/utils"
	"os"
	"path/filepath"
)

const Image = "softwaremill/elasticmq:1.3.4"

func Container(cfg *settings.Config) (*docker.Container, error) {
	basePath := filepath.Join(cfg.DataPath(), "sqs")
	configPath := filepath.Join(cfg.DataPath(), "sqs.conf")

	err := utils.CreateDirs(basePath)
	if err != nil {
		msg := log.Error("Unable to create directory %s: %v", basePath, err)
		return nil, errors.New(msg)
	}

	err = writeConfigFile(cfg, configPath)
	if err != nil {
		msg := log.Error("Unable to write SQS config file: %v", err)
		return nil, errors.New(msg)
	}

	return &docker.Container{
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
			9324: cfg.SQS.Port,
			9325: cfg.SQS.Port + 1,
		},
	}, nil
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

func writeConfigFile(cfg *settings.Config, configPath string) error {
	stat, err := os.Stat(configPath)
	if err == nil && stat.IsDir() {
		msg := log.Error("Expecting %s to be a file, but is a directory: %v", configPath, err)
		return errors.New(msg)
	}

	if err == nil {
		log.Info("The file %s already exists, so returning without creating.")
		return nil
	}

	f, err := os.Create(configPath)
	if err != nil {
		msg := log.Error("Unable to open %s: %v", configPath, err)
		return errors.New(msg)
	}

	contents := fmt.Sprintf(configFileTemplate, cfg.Region, cfg.AccountNumber)
	_, err = f.WriteString(contents)
	if err != nil {
		msg := log.Error("Unable to write contents to %s: %v", configPath, err)
		return errors.New(msg)
	}

	return nil
}

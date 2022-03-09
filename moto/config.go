package moto

import (
	"errors"
	"github.com/docker/docker/api/types/mount"
	"myaws/docker"
	"myaws/log"
	"myaws/settings"
	"myaws/utils"
	"path/filepath"
)

const Image = "motoserver/moto:3.0.4"

func Container(cfg *settings.Config) (*docker.Container, error) {
	basePath := filepath.Join(cfg.DataPath(), "moto")
	err := utils.CreateDirs(basePath)
	if err != nil {
		msg := log.Error("Unable to create directory %s: %v", basePath, err)
		return nil, errors.New(msg)
	}

	return &docker.Container{
		Name:  "moto",
		Image: Image,
		Mounts: []mount.Mount{
			{
				Source: basePath,
				Target: "/data",
				Type:   mount.TypeBind,
			},
		},
		Ports: map[int]int{
			5000: cfg.Moto.Port,
		},
	}, nil
}

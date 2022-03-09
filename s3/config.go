package s3

import (
	"errors"
	"github.com/docker/docker/api/types/mount"
	"myaws/docker"
	"myaws/log"
	"myaws/settings"
	"myaws/utils"
	"path/filepath"
)

const Image = "bitnami/minio:2022.2.16"

func Container(cfg *settings.Config) (*docker.Container, error) {
	basePath := filepath.Join(cfg.DataPath(), "s3")
	err := utils.CreateDirs(basePath)
	if err != nil {
		msg := log.Error("Unable to create directory %s: %v", basePath, err)
		return nil, errors.New(msg)
	}

	return &docker.Container{
		Name:  "s3",
		Image: Image,
		Mounts: []mount.Mount{
			{
				Source: basePath,
				Target: "/data",
				Type:   mount.TypeBind,
			},
		},
		Ports: map[int]int{
			9000: cfg.S3.Port,
			9001: cfg.S3.Port + 1,
		},
	}, nil
}

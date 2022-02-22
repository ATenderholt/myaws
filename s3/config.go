package s3

import (
	"github.com/docker/docker/api/types/mount"
	"myaws/config"
	"myaws/docker"
	"myaws/log"
	"myaws/utils"
	"path/filepath"
)

const Image = "bitnami/minio:2022.2.16"

var basePath = filepath.Join(config.GetDataPath(), "s3")
var Container = docker.Container{
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
		9000: 9000,
		9001: 9001,
	},
}

func init() {
	err := utils.CreateDirs(basePath)
	if err != nil {
		msg := log.Error("Unable to create directory %s: %v", basePath, err)
		panic(msg)
	}
}

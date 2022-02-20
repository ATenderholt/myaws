package s3

import (
	"github.com/docker/docker/api/types/mount"
	"myaws/config"
	"myaws/docker"
	"path/filepath"
)

const Image = "bitnami/minio:2022.2.16"

var Container = docker.Container{
	Name:  "s3",
	Image: Image,
	Mounts: []mount.Mount{
		{
			Source: filepath.Join(config.GetSettings().GetDataPath(), "s3"),
			Target: "/data",
			Type:   mount.TypeBind,
		},
	},
	Ports: map[int]int{
		9000: 9000,
		9001: 9001,
	},
}

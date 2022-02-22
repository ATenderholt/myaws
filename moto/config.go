package moto

import (
	"github.com/docker/docker/api/types/mount"
	"myaws/config"
	"myaws/docker"
	"myaws/log"
	"myaws/utils"
	"path/filepath"
)

const Image = "motoserver/moto:3.0.4"

var basePath = filepath.Join(config.GetDataPath(), "moto")
var Container = docker.Container{
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
		5000: config.Moto().Port,
	},
}

func init() {
	err := utils.CreateDirs(basePath)
	if err != nil {
		msg := log.Error("Unable to create directory %s: %v", basePath, err)
		panic(msg)
	}
}

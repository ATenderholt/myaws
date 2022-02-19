package docker

import (
	"github.com/docker/docker/api/types/mount"
)

type Container struct {
	Name   string
	Image  string
	id     string
	Mounts []mount.Mount
}

package docker

import (
	"fmt"
	"github.com/docker/docker/api/types/mount"
)

type Container struct {
	Name    string
	Image   string
	ID      string
	Mounts  []mount.Mount
	command []string
}

func (c Container) String() string {
	return fmt.Sprintf("%s (%s/%s)", c.Name, c.Image, c.ID)
}

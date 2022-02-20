package docker

import (
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"myaws/log"
	"os"
)

type Container struct {
	Name        string
	Image       string
	ID          string
	Mounts      []mount.Mount
	Ports       map[int]int
	Command     []string
	Environment []string
}

func (c Container) String() string {
	return fmt.Sprintf("%s (%s/%s)", c.Name, c.Image, c.ID)
}

func (c Container) PortBindings() (map[nat.Port]struct{}, map[nat.Port][]nat.PortBinding, error) {
	specs := make([]string, len(c.Ports))
	i := 0
	for from, to := range c.Ports {
		// ip:public:private/proto
		specs[i] = fmt.Sprintf("0.0.0.0:%d:%d/tcp", to, from)
		i += 1
	}

	return nat.ParsePortSpecs(specs)
}

func (c Container) GetMounts() []mount.Mount {
	for _, mnt := range c.Mounts {
		dest := mnt.Source
		stats, err := os.Stat(dest)

		if err == nil && stats.IsDir() {
			continue
		}

		if err == nil && !stats.IsDir() {
			msg := log.Error("%s already exists, but is not a directory", dest)
			panic(msg)
		}

		if errors.Is(err, os.ErrNotExist) {
			log.Info("Creating directory %s ...", dest)
			err2 := os.MkdirAll(dest, 0755)
			if err2 != nil {
				msg := log.Error("Unable to create %s: %v", dest, err2)
				panic(msg)
			}
			log.Info(".... %s created.", dest)
		}
	}

	return c.Mounts
}

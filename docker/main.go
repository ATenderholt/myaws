package docker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"myaws/log"
	"myaws/utils"
)

type Docker struct {
	cli     *client.Client
	running map[string]Container
}

func NewController() *Docker {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	running := make(map[string]Container, 5)
	return &Docker{cli: cli, running: running}
}

func (d *Docker) EnsureImage(image string) {
	reader, err := d.cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		msg := log.Error("Error when ensuring image %s exists: %v", image, err)
		panic(msg)
	}

	defer reader.Close()
	lines := utils.ReadLinesAsBytes(reader)
	for line := range lines {
		var progress EnsureImageProgress
		err := json.Unmarshal(line, &progress)
		if err != nil {
			log.Info("[DOCKER] %s", string(line))
			continue
		}

		log.Info("[DOCKER] %s", progress)
	}
}

func (d *Docker) Start(c Container) error {
	portSet, portMap, err := c.PortBindings()
	if err != nil {
		msg := log.Error("Unable to get port bindings: %v", err)
		return errors.New(msg)
	}

	hostConfig := container.HostConfig{}
	hostConfig.Mounts = c.Mounts
	hostConfig.PortBindings = portMap

	containerConfig := container.Config{
		ExposedPorts: portSet,
		Tty:          false,
		Cmd:          c.Command,
		Image:        c.Image,
	}

	ctx := context.Background()
	resp, err := d.cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, c.Name)
	if err != nil {
		msg := log.Error("Unable to create container %s: %v", c, err)
		return errors.New(msg)
	}

	err = d.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		msg := log.Error("Unable to start container %s: %v", c, err)
		return errors.New(msg)
	}

	c.ID = resp.ID
	d.running[c.Name] = c

	go func() {
		logOptions := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true}
		reader, err := d.cli.ContainerLogs(ctx, resp.ID, logOptions)

		if err != nil {
			log.Error("Unable to follow logs for container %s: %v", c, err)
			return
		}
		defer reader.Close()

		lines := utils.ReadLinesAsBytes(reader)
		for line := range lines {
			log.Info("[DOCKER %s] %s", c.Name, string(line))
		}

		log.Info("[DOCKER] Logs finished for container %s", c)
	}()

	return nil
}

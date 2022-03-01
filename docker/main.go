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
	"strings"
	"time"
)

var instance *Docker

func init() {
	instance = NewController()
}

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

func EnsureImage(ctx context.Context, image string) {
	reader, err := instance.cli.ImagePull(ctx, image, types.ImagePullOptions{})
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

func Start(ctx context.Context, c Container) error {
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
		Env:          c.Environment,
	}

	resp, err := instance.cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, c.Name)
	if err != nil {
		msg := log.Error("Unable to create container %s: %v", c, err)
		return errors.New(msg)
	}

	err = instance.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		msg := log.Error("Unable to start container %s: %v", c, err)
		return errors.New(msg)
	}

	c.ID = resp.ID
	instance.running[c.Name] = c

	go func() {
		logOptions := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true}
		// logs need to be in backgroud context so they aren't canceled before container.
		reader, err := instance.cli.ContainerLogs(context.Background(), resp.ID, logOptions)

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

func Shutdown(ctx context.Context, c Container) error {
	log.Info("Trying to shutdown %s...", c)

	timeout := 30 * time.Second
	err := instance.cli.ContainerStop(ctx, c.ID, &timeout)
	if err != nil {
		msg := log.Error("Unable to shutdown container %s: %v", c, err)
		return errors.New(msg)
	}

	delete(instance.running, c.Name)

	err = instance.cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{})
	if err != nil {
		msg := log.Error("Unable to remove container %s: %v", c, err)
		return errors.New(msg)
	}

	return nil
}

func ShutdownAll(ctx context.Context) error {
	var allErrors []string
	for _, c := range instance.running {
		err := Shutdown(ctx, c)
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	msg := strings.Join(allErrors, ",")
	if len(msg) > 0 {
		return errors.New(msg)
	}

	return nil
}

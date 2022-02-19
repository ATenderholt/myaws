package docker

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"myaws/log"
	"myaws/utils"
)

type Docker struct {
	cli *client.Client
}

func NewController() *Docker {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	return &Docker{cli}
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

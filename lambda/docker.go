package lambda

import (
	"errors"
	"github.com/docker/docker/api/types/mount"
	"myaws/config"
	"myaws/docker"
	"myaws/lambda/types"
	"myaws/log"
)

type PortPool struct {
	available map[int]bool
}

var pool PortPool

func init() {
	pool = NewPortPool(config.Lambda().Port, config.Lambda().Port+100)
}

func NewPortPool(min, max int) PortPool {
	available := make(map[int]bool, max-min)

	pool := PortPool{available}
	for i := min; i <= max; i++ {
		pool.available[i] = true
	}

	return pool
}

func (pool PortPool) Get() (int, error) {
	var result = -1
	for port, available := range pool.available {
		if available {
			result = port
			break
		}
	}

	if result == -1 {
		log.Error("No ports are available")
		return result, nil
	}

	pool.available[result] = false
	return result, nil
}

func StartFunction(function *types.Function) error {
	port, err := pool.Get()
	if err != nil {
		msg := log.Error("Unable to start Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}

	log.Info("Starting Function %s on port %d", function.FunctionName, port)

	container := docker.Container{
		Name:  function.FunctionName,
		Image: "mlupin/docker-lambda:" + string(function.Runtime),
		Mounts: []mount.Mount{
			{
				Source:      function.GetDestPath(),
				Target:      "/var/task",
				Type:        mount.TypeBind,
				ReadOnly:    true,
				Consistency: mount.ConsistencyDelegated,
			},
			{
				Source:      function.GetLayerDestPath(),
				Target:      "/opt",
				Type:        mount.TypeBind,
				ReadOnly:    true,
				Consistency: mount.ConsistencyDelegated,
			},
		},
		Environment: []string{
			"DOCKER_LAMBDA_STAY_OPEN=1",
		},
		Ports: map[int]int{
			9001: port,
		},
	}

	err = docker.Start(container)
	if err != nil {
		msg := log.Error("Unable to start Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}

	return nil
}

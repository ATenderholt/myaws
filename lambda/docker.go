package lambda

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"io"
	"myaws/config"
	"myaws/docker"
	"myaws/lambda/types"
	"myaws/log"
	"net/http"
)

var manager *ManagerImpl

type Manager interface {
	Add(name string, port int)
	Invoke(response http.ResponseWriter, request *http.Request)
}

type ManagerImpl struct {
	ports map[string]int
}

func (manager *ManagerImpl) Add(name string, port int) {
	manager.ports[name] = port
}

func (manager *ManagerImpl) Invoke(name string, response *http.ResponseWriter, request *http.Request) {
	log.Info("Invoking Function %s ...", name)

	url := fmt.Sprintf("http://%s:%d%s", "localhost", manager.ports[name], request.URL.Path)

	//var proxyRequestBody strings.Builder
	//requestBody := io.TeeReader(request.Body, &proxyRequestBody)
	//authorization := request.Header.Get("Authorization")
	//contentType := request.Header.Get("Content-Type")
	//target := request.Header.Get("X-Amz-Target")

	proxyReq, _ := http.NewRequest(request.Method, url, request.Body)
	//proxyReq.Header.Set("Content-Type", contentType)
	//proxyReq.Header.Set("Authorization", authorization)
	//if len(target) > 0 {
	//	proxyReq.Header.Set("X-Amz-Target", target)
	//}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := log.Error("... unable to invoke %s: %v", name, err)
		http.Error(*response, msg, http.StatusInternalServerError)
		return
	}

	log.Debug("Got following response when invoking Function %s: %+v", name, resp)

	for key, value := range resp.Header {
		for _, v := range value {
			(*response).Header().Add(key, v)
		}

	}

	(*response).WriteHeader(resp.StatusCode)

	io.Copy(*response, resp.Body)
	resp.Body.Close()
}

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

func StartFunction(ctx context.Context, function *types.Function) error {
	port, err := pool.Get()
	if err != nil {
		msg := log.Error("Unable to start Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}

	log.Info("Starting Function %s on port %d using handler %s", function.FunctionName, port, function.Handler)

	container := docker.Container{
		Name:    function.FunctionName,
		Image:   "mlupin/docker-lambda:" + string(function.Runtime),
		Command: []string{function.Handler},
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

	err = docker.Start(ctx, container)
	if err != nil {
		msg := log.Error("Unable to start Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}

	if manager == nil {
		manager = &ManagerImpl{ports: make(map[string]int)}
	}

	manager.Add(function.FunctionName, port)

	return nil
}

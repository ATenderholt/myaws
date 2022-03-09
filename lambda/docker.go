package lambda

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/docker/distribution/uuid"
	"github.com/docker/docker/api/types/mount"
	"io"
	"myaws/docker"
	"myaws/lambda/types"
	"myaws/log"
	"myaws/settings"
	"net/http"
	"strings"
)

var manager *ManagerImpl

type Manager interface {
	Add(name string, port int)
	Invoke(response http.ResponseWriter, request *http.Request)
	StartEventSource(ctx context.Context, eventSource *types.EventSource)
}

type ManagerImpl struct {
	ports        map[string]int
	eventSources map[uuid.UUID]context.CancelFunc
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

var credentials aws.CredentialsProviderFunc = func(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{AccessKeyID: "", SecretAccessKey: "", CanExpire: false}, nil
}

var endpointResolver aws.EndpointResolverWithOptionsFunc = func(service, region string, options ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL:               "http://localhost:9324",
		HostnameImmutable: true,
	}, nil
}

func StartEventSource(ctx context.Context, eventSource *types.EventSource) error {
	return manager.StartEventSource(ctx, eventSource)
}

func (manager *ManagerImpl) StartEventSource(ctx context.Context, eventSource *types.EventSource) error {
	parts := strings.Split(eventSource.Arn, ":")
	queueName := parts[5]

	log.Info("Starting consumption from Queue %s ...", queueName)

	runCtx, cancel := context.WithCancel(ctx)
	cfg := aws.Config{
		Region:                      "us-west-2",
		Credentials:                 credentials,
		HTTPClient:                  nil,
		EndpointResolver:            nil,
		EndpointResolverWithOptions: endpointResolver,
		RetryMaxAttempts:            0,
		RetryMode:                   "",
		Retryer:                     nil,
		ConfigSources:               nil,
		APIOptions:                  nil,
		Logger:                      nil,
		ClientLogMode:               0,
		DefaultsMode:                "",
		RuntimeEnvironment:          aws.RuntimeEnvironment{},
	}

	client := sqs.NewFromConfig(cfg)

	listQueuesOutput, err := client.ListQueues(ctx, &sqs.ListQueuesInput{QueueNamePrefix: &queueName})
	if err != nil {
		msg := log.Error("Unable to list queues for %s: %v", queueName, err)
		cancel()
		return errors.New(msg)
	}

	if len(listQueuesOutput.QueueUrls) != 1 {
		msg := log.Error("Found %d queue urls for %s: %v", len(listQueuesOutput.QueueUrls), queueName, listQueuesOutput.QueueUrls)
		cancel()
		return errors.New(msg)
	}

	queueUrl := listQueuesOutput.QueueUrls[0]
	receiveMessageInput := sqs.ReceiveMessageInput{
		QueueUrl:                &queueUrl,
		AttributeNames:          nil,
		MaxNumberOfMessages:     eventSource.BatchSize,
		MessageAttributeNames:   nil,
		ReceiveRequestAttemptId: nil,
		VisibilityTimeout:       0,
		WaitTimeSeconds:         5,
	}

	go func() {
		for {
			select {
			case <-runCtx.Done():
				return
			default:
				receiveMessageOutput, err := client.ReceiveMessage(ctx, &receiveMessageInput)
				if err != nil {
					log.Error("Error: %v", err)
					continue
				}

				if len(receiveMessageOutput.Messages) == 0 {
					log.Info("No messages")
				}
				for _, message := range receiveMessageOutput.Messages {
					log.Info("Received %+v", message)
				}
			}
		}
	}()

	manager.eventSources[eventSource.UUID] = cancel

	return nil
}

type PortPool struct {
	available map[int]bool
}

var pool *PortPool

func NewPortPool(min, max int) *PortPool {
	available := make(map[int]bool, max-min)

	pool := PortPool{available}
	for i := min; i <= max; i++ {
		pool.available[i] = true
	}

	return &pool
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
	cfg := settings.FromContext(ctx)
	if pool == nil {
		pool = NewPortPool(cfg.Lambda.Port, cfg.Lambda.Port+100)
	}

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
				Source:      function.GetDestPath(ctx),
				Target:      "/var/task",
				Type:        mount.TypeBind,
				ReadOnly:    true,
				Consistency: mount.ConsistencyDelegated,
			},
			{
				Source:      function.GetLayerDestPath(ctx),
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

	_, err = docker.Start(ctx, container, "")
	if err != nil {
		msg := log.Error("Unable to start Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}

	if manager == nil {
		manager = &ManagerImpl{ports: make(map[string]int), eventSources: make(map[uuid.UUID]context.CancelFunc)}
	}

	manager.Add(function.FunctionName, port)

	return nil
}

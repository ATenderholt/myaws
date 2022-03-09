package sqs

import (
	"errors"
	"io"
	"myaws/log"
	"myaws/settings"
	"net/http"
	"strings"
)

func ProxyToElasticMQ(response http.ResponseWriter, request *http.Request) {
	cfg := settings.FromContext(request.Context())
	in, out, err := proxyToElasticMQ(&response, request, cfg.Region)
	log.Info("SQS Request Payload: %s", in)
	log.Info("SQS Response Body: %s", out)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func proxyToElasticMQ(response *http.ResponseWriter, request *http.Request, region string) (in string, out string, err error) {
	cfg := settings.FromContext(request.Context())
	url := cfg.SQS.BuildUrl(request.URL.Path)

	var payloadBuilder strings.Builder
	requestBody := io.TeeReader(request.Body, &payloadBuilder)
	proxyReq, _ := http.NewRequest(request.Method, url, requestBody)
	proxyReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := log.Error("Problem proxying to ElasticMQ: %v", err)
		return payloadBuilder.String(), "", errors.New(msg)
	}

	log.Info("Got following response from ElasticMQ: %+v", resp)

	// have to peek inside request since ElasticMQ doesn't seem to support /persist same properties that Terraform expects
	// and do different things
	payload := payloadBuilder.String()
	for action, f := range actions {
		if strings.Contains(payload, action) {
			log.Info("Doing extra work for %s ...", action)
			return f(request.Context(), response, resp, payload)
		}
	}

	var responseBuilder strings.Builder
	responseBody := io.TeeReader(resp.Body, &responseBuilder)

	(*response).WriteHeader(resp.StatusCode)
	io.Copy(*response, responseBody)
	resp.Body.Close()

	return payload, responseBuilder.String(), nil
}

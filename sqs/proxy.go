package sqs

import (
	"errors"
	"fmt"
	"io"
	"myaws/config"
	"myaws/log"
	"net/http"
)

func ProxyToElasticMQ(response http.ResponseWriter, request *http.Request) {
	err := proxyToElasticMQ(&response, request, config.Region())
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func proxyToElasticMQ(response *http.ResponseWriter, request *http.Request, region string) error {
	url := fmt.Sprintf("http://%s:%d%s", config.SQS().Host, config.SQS().Port, request.URL.Path)
	proxyReq, _ := http.NewRequest(request.Method, url, request.Body)
	proxyReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := log.Error("Problem proxying to ElasticMQ: %v", err)
		return errors.New(msg)
	}

	log.Info("Got following response from ElasticMQ: %+v", resp)

	for key, value := range resp.Header {
		for _, v := range value {
			(*response).Header().Add(key, v)
		}

	}

	(*response).WriteHeader(resp.StatusCode)
	io.Copy(*response, resp.Body)
	resp.Body.Close()

	return nil
}

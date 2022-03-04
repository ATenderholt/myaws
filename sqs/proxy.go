package sqs

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"myaws/config"
	"myaws/log"
	"net/http"
	"strings"
)

type QueueAttribute struct {
	Name  string
	Value string
}

type GetQueueAttributesResult struct {
	Attribute []QueueAttribute
}

type ResponseMetadata struct {
	RequestId string
}

type GetQueueAttributesResponse struct {
	GetQueueAttributesResult GetQueueAttributesResult
	ResponseMetadata         ResponseMetadata
}

func ProxyToElasticMQ(response http.ResponseWriter, request *http.Request) {
	in, out, err := proxyToElasticMQ(&response, request, config.Region())
	log.Info("SQS Request Payload: %s", in)
	log.Info("SQS Response Body: %s", out)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func proxyToElasticMQ(response *http.ResponseWriter, request *http.Request, region string) (in string, out string, err error) {
	url := fmt.Sprintf("http://%s:%d%s", config.SQS().Host, config.SQS().Port, request.URL.Path)

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

	for key, value := range resp.Header {
		for _, v := range value {
			(*response).Header().Add(key, v)
		}
	}

	// have to peek inside request since ElasticMQ doesn't seem to support /persist same properties that Terraform expects
	// and do different things
	payload := payloadBuilder.String()
	if strings.Contains(payload, "GetQueueAttributes") {
		log.Info("Handling %s", payload)
		parser := xml.NewDecoder(resp.Body)
		var output GetQueueAttributesResponse
		err := parser.Decode(&output)
		if err != nil {
			msg := log.Error("Unable to unmarshall %s: %v", payload, err)
			return payload, "", errors.New(msg)
		}

		log.Info("Got following response object: %+v", output)
		attributes := output.GetQueueAttributesResult.Attribute
		attributes = append(attributes, QueueAttribute{Name: "MaximumMessageSize", Value: "262144"})
		attributes = append(attributes, QueueAttribute{Name: "MessageRetentionPeriod", Value: "345600"})
		output.GetQueueAttributesResult.Attribute = attributes

		(*response).WriteHeader(resp.StatusCode)
		b, err := xml.Marshal(output)
		if err != nil {
			msg := log.Error("Unable to marshall %v: %v", output, err)
			return payload, "", errors.New(msg)
		}
		io.Copy(*response, bytes.NewReader(b))
		resp.Body.Close()

		return payload, "", nil

	}

	var responseBuilder strings.Builder
	responseBody := io.TeeReader(resp.Body, &responseBuilder)

	(*response).WriteHeader(resp.StatusCode)
	io.Copy(*response, responseBody)
	resp.Body.Close()

	return payload, responseBuilder.String(), nil
}

package sqs

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"myaws/database"
	"myaws/log"
	"myaws/settings"
	"myaws/sqs/queries"
	"myaws/sqs/types"
	"net/http"
	"regexp"
	"strings"
)

var queueNameRegex *regexp.Regexp
var queueUrlRegex *regexp.Regexp
var createQueueAttributeRegex *regexp.Regexp
var createQueueTagRegex *regexp.Regexp
var actions map[string]types.ExtraWorkFunction

func init() {
	var err error
	queueNameRegex, err = regexp.Compile(`QueueName=([^&]*)`)
	if err != nil {
		panic("unable to compile queue name regex")
	}

	// name of queue is everything after trailing slash
	queueUrlRegex, err = regexp.Compile(`QueueUrl=.*%2F([^&]+)`)
	if err != nil {
		panic("unable to compile queue url regex")
	}

	createQueueAttributeRegex, err = regexp.Compile(`Attribute.\d+.Name=([^&]*)&Attribute.\d+.Value=([^&]*)`)
	if err != nil {
		panic("unable to compile create queue attribute regex")
	}

	createQueueTagRegex, err = regexp.Compile(`Tag.\d+.Key=([^&]*)&Tag.\d+.Value=([^&]*)`)
	if err != nil {
		panic("unable to compile create queue tag regex")
	}

	actions = make(map[string]types.ExtraWorkFunction)
	actions["CreateQueue"] = createQueue
	actions["GetQueueAttributes"] = getQueueAttributes
}

func parsePayload(payload string) (map[string]string, error) {
	log.Debug("Parsing '%s' ... ", payload)

	result := make(map[string]string)

	parts := strings.Split(payload, "&")
	for _, part := range parts {
		pieces := strings.Split(part, "=")
		if len(pieces) != 2 {
			msg := log.Error("Unexpected number of pieces (%d) for %s", len(pieces), part)
			return nil, errors.New(msg)
		}

		result[pieces[0]] = pieces[1]
	}

	return result, nil
}

func createQueue(ctx context.Context, writer *http.ResponseWriter, proxyResponse *http.Response, payload string) (string, string, error) {
	defer proxyResponse.Body.Close()

	name := queueNameRegex.FindStringSubmatch(payload)
	if name == nil {
		msg := log.Error("unable to find queue name in %s", payload)
		return payload, "", errors.New(msg)
	}

	attributes := createQueueAttributeRegex.FindAllStringSubmatch(payload, -1)
	if attributes == nil {
		msg := log.Error("unable to find attributes in %s", payload)
		return payload, "", errors.New(msg)
	}

	tags := createQueueTagRegex.FindAllStringSubmatch(payload, -1)
	if tags == nil {
		msg := log.Error("unable to find tags in %s", payload)
		return payload, "", errors.New(msg)
	}

	queue := types.NewQueue(name[1])

	for _, matches := range attributes {
		queue.Attributes[matches[1]] = matches[2]
	}

	for _, matches := range attributes {
		queue.Tags[matches[1]] = matches[2]
	}

	cfg := settings.FromContext(ctx)
	db := database.CreateConnection(cfg)
	defer db.Close()

	err := queries.SaveQueue(ctx, db, queue)
	if err != nil {
		return payload, "", err
	}

	var responseBuilder strings.Builder
	responseBody := io.TeeReader(proxyResponse.Body, &responseBuilder)

	(*writer).WriteHeader(proxyResponse.StatusCode)
	io.Copy(*writer, responseBody)

	return payload, responseBuilder.String(), nil
}

func getQueueAttributes(ctx context.Context, writer *http.ResponseWriter, proxyResponse *http.Response, payload string) (string, string, error) {
	defer proxyResponse.Body.Close()

	name := queueUrlRegex.FindStringSubmatch(payload)
	if name == nil {
		msg := log.Error("unable to find queue url in %s", payload)
		return payload, "", errors.New(msg)
	}

	cfg := settings.FromContext(ctx)
	db := database.CreateConnection(cfg)
	defer db.Close()

	queue, err := queries.LoadQueue(ctx, db, name[1])
	if err != nil {
		msg := log.Error("unable to load queue %s: %v", name[1], err)
		return payload, "", errors.New(msg)
	}

	// queue doesn't exist
	if len(queue.Attributes) == 0 {
		var responseBuilder strings.Builder
		responseBody := io.TeeReader(proxyResponse.Body, &responseBuilder)

		(*writer).WriteHeader(proxyResponse.StatusCode)
		io.Copy(*writer, responseBody)
		return payload, responseBuilder.String(), nil
	}

	parser := xml.NewDecoder(proxyResponse.Body)
	var output types.GetQueueAttributesResponse
	err = parser.Decode(&output)
	if err != nil {
		msg := log.Error("Unable to unmarshall %s: %v", payload, err)
		return payload, "", errors.New(msg)
	}

	log.Info("Got following response object: %+v", output)

	for key, value := range queue.Attributes {
		output.AddAttributeIfNotExists(key, value)
	}

	(*writer).WriteHeader(proxyResponse.StatusCode)
	b, err := xml.Marshal(output)
	if err != nil {
		msg := log.Error("Unable to marshall %v: %v", output, err)
		return payload, "", errors.New(msg)
	}
	io.Copy(*writer, bytes.NewReader(b))

	return payload, string(b), nil
}

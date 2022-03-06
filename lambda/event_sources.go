package lambda

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/docker/distribution/uuid"
	"io"
	"myaws/database"
	"myaws/lambda/queries"
	"myaws/lambda/types"
	"myaws/log"
	"myaws/utils"
	"net/http"
	"strings"
	"time"
)

const PostEventSourceRegex = `^/2015-03-31/event-source-mappings/$`

func PostEventSource(writer http.ResponseWriter, request *http.Request) {
	var requestBodyBuilder strings.Builder
	reader := io.TeeReader(request.Body, &requestBodyBuilder)

	var payload lambda.CreateEventSourceMappingInput
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&payload)
	if err != nil {
		msg := log.Error("unable to decode body for creating an Event Source: %v", err)
		http.Error(writer, msg, http.StatusInternalServerError)
		return
	}

	ctx := request.Context()
	db := database.CreateConnection()

	function, err := queries.LatestFunctionByName(ctx, db, *payload.FunctionName)
	if err != nil {
		msg := log.Error("unable to load Function %s: %v", payload.FunctionName, err)
		http.Error(writer, msg, http.StatusInternalServerError)
		return
	}

	eventSource := types.EventSource{
		UUID:         uuid.Generate(),
		Enabled:      true,
		Arn:          *payload.EventSourceArn,
		Function:     function,
		BatchSize:    *payload.BatchSize,
		LastModified: time.Now().UnixMilli(),
	}

	log.Info("Saving Event Source: %+v", eventSource)

	err = queries.SaveEventSource(ctx, db, eventSource)
	if err != nil {
		msg := log.Error("unable to save Event Source %+v: %v", eventSource, err)
		http.Error(writer, msg, http.StatusInternalServerError)
		return
	}

	body := eventSource.ToCreateEventSourceMappingOutput()

	utils.RespondWithJson(writer, body)
}

const GetEventSourceRegex = `^/2015-03-31/event-source-mappings/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`

func GetEventSource(writer http.ResponseWriter, request *http.Request) {
	parts := strings.Split(request.URL.Path, "/")
	id := parts[3]

	log.Info("Getting event source %s ... ", id)

	ctx := request.Context()
	db := database.CreateConnection()

	eventSource, err := queries.LoadEventSource(ctx, db, id)
	if err != nil {
		msg := log.Error("Unable to load Event Source %+v: %v", eventSource, err)
		http.Error(writer, msg, http.StatusInternalServerError)
		return
	}

	if eventSource == nil {
		log.Info("Event Source %s not found", id)
		http.NotFound(writer, request)
		return
	}
	
	body := eventSource.ToGetEventSourceMappingOutput()

	utils.RespondWithJson(writer, body)
}

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
		msg := log.Error("unable to decode body for creating an event source: %v", err)
		http.Error(writer, msg, http.StatusInternalServerError)
		return
	}

	ctx := request.Context()
	db := database.CreateConnection()

	function, err := queries.LatestFunctionByName(ctx, db, *payload.FunctionName)
	if err != nil {
		msg := log.Error("error when loading function %s: %v", payload.FunctionName, err)
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

	log.Info("Saving event source: %+v", eventSource)

	err = queries.SaveEventSource(ctx, db, eventSource)
	if err != nil {
		msg := log.Error("unable to save event source %+v: %v", eventSource, err)
		http.Error(writer, msg, http.StatusInternalServerError)
		return
	}

	body := eventSource.ToCreateEventSourceMappingOutput()

	utils.RespondWithJson(writer, body)
}

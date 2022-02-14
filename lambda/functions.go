package lambda

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"myaws/database"
	"myaws/lambda/queries"
	"myaws/lambda/types"
	"myaws/log"
	"myaws/utils"
	"net/http"
	"strconv"
)

const PostLambdaFunctionRegex = `^/2015-03-31/functions$`

func PostLambdaFunction(response http.ResponseWriter, request *http.Request) {
	dec := json.NewDecoder(request.Body)

	var body lambda.CreateFunctionInput
	err := dec.Decode(&body)

	if err != nil {
		msg := fmt.Sprintf("error decoding %s: %v", request.Body, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	log.Info("Creating lambda function %s", body.FunctionName)

	ctx := request.Context()
	db := database.CreateConnection()
	defer db.Close()

	runtimeExists, err := queries.RuntimeExistsByName(ctx, db, body.Runtime)
	if err != nil {
		msg := log.Error("Error when querying runtime %s for function %s", body.Runtime, body.FunctionName)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	if !runtimeExists {
		msg := log.Error("Unable to find runtime %s for function %s", body.Runtime, body.FunctionName)
		http.Error(response, msg, http.StatusNotFound)
		return
	}

	dbVersion, err := queries.LatestFunctionVersionByName(ctx, db, body.FunctionName)
	if err != nil {
		msg := log.Error("Error when finding latest version of function %s", body.FunctionName)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	version := strconv.Itoa(dbVersion + 1)

	function := types.Function{
		FunctionName:  *body.FunctionName,
		Description:   *body.Description,
		Handler:       *body.Handler,
		Role:          *body.Role,
		DeadLetterArn: *body.DeadLetterConfig.TargetArn,
		LayerArns:     body.Layers,
		MemorySize:    *body.MemorySize,
		Runtime:       body.Runtime,
		Timeout:       *body.Timeout,
		Version:       &version,
		Environment:   body.Environment,
		Tags:          body.Tags,
	}

	saved, err := queries.InsertFunction(ctx, db, &function)
	result := saved.ToCreateFunctionOutput()

	utils.RespondWithJson(response, result)
}
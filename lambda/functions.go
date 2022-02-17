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
	"strings"
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

	// move Code to new variable so that most of response body can be printed
	code := body.Code
	body.Code = nil
	log.Info("Creating lambda function %+v", body)

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

	function := types.CreateFunction(&body)
	function.Version = strconv.Itoa(dbVersion + 1)

	err = utils.DecompressZipFile(code.ZipFile, function.GetDestPath())
	if err != nil {
		msg := fmt.Sprintf("error when saving function %s: %v", *body.FunctionName, err)
		log.Error(msg)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	saved, err := queries.InsertFunction(ctx, db, function)
	result := saved.ToCreateFunctionOutput()

	utils.RespondWithJson(response, result)
}

func getFunctionName(path string) string {
	parts := strings.Split(path, "/")
	return parts[3]
}

const GetLambdaFunctionRegex = `^/2015-03-31/functions/[A-Za-z0-9_-]+$`

func GetLambdaFunction(response http.ResponseWriter, request *http.Request) {
	name := getFunctionName(request.URL.Path)

	log.Info("Getting Lambda Function %s", name)

	ctx := request.Context()
	db := database.CreateConnection()
	defer db.Close()

	function, err := queries.FunctionByName(ctx, db, name)
	if err != nil {
		msg := log.Error("Unable to get Lambda Function %s: %v", name, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	result := function.ToGetFunctionOutput()

	utils.RespondWithJson(response, result)
}

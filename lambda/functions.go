package lambda

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go/middleware"
	"myaws/database"
	"myaws/lambda/queries"
	"myaws/lambda/types"
	"myaws/log"
	"myaws/settings"
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
	cfg := settings.FromContext(ctx)
	db := database.CreateConnection(cfg)
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
	rawHash := sha256.Sum256(code.ZipFile)
	function.CodeSha256 = base64.StdEncoding.EncodeToString(rawHash[:])

	// TODO : validate Layer runtime support

	err = utils.UncompressZipFileBytes(code.ZipFile, function.GetDestPath(ctx))
	if err != nil {
		msg := fmt.Sprintf("error when saving function %s: %v", *body.FunctionName, err)
		log.Error(msg)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	layerDestPath := function.GetLayerDestPath(ctx)
	err = utils.CreateDirs(layerDestPath)
	if err != nil {
		msg := log.Error("Unable to create Layer path for Function %s: %v", function.FunctionName, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	for _, layer := range function.Layers {
		layerPath := layer.GetDestPath(ctx)
		err = utils.UncompressZipFile(layerPath, layerDestPath)
		if err != nil {
			msg := log.Error("error when unpacking layer %s: %v", layer.Name, err)
			http.Error(response, msg, http.StatusInternalServerError)
			return
		}
	}

	saved, err := queries.InsertFunction(ctx, db, function)
	result := saved.ToCreateFunctionOutput(ctx)

	utils.RespondWithJson(response, result)
}

func getFunctionName(path string) string {
	parts := strings.Split(path, "/")
	return parts[3]
}

const PutLambdaConfigurationRegex = "^/2015-03-31/functions/[A-Za-z0-9_-]+/configuration$"

func PutLambdaConfiguration(response http.ResponseWriter, request *http.Request) {
	name := getFunctionName(request.URL.Path)

	log.Info("Setting configuration for Lambda Function %s ...", name)

	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()

	var body lambda.UpdateFunctionConfigurationInput
	err := decoder.Decode(&body)
	if err != nil {
		msg := log.Error("Error when decoding body: %v", err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	log.Info("Configuration: %+v", body)

	ctx := request.Context()
	cfg := settings.FromContext(ctx)
	db := database.CreateConnection(cfg)
	defer db.Close()

	function, err := queries.LatestFunctionByName(ctx, db, name)

	switch {
	case err == sql.ErrNoRows:
		log.Info("Unable to find Function named %s", name)
		http.NotFound(response, request)
		return
	case err != nil:
		msg := log.Error("Error when querying for Function %s: %v", name, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	if body.Environment != nil {
		err = queries.UpsertFunctionEnvironment(ctx, db, function, body.Environment)
		if err != nil {
			msg := log.Error("Error when upserting Environment for Function %s: %v", name, err)
			http.Error(response, msg, http.StatusInternalServerError)
			return
		}
	}

	result := function.ToUpdateFunctionConfigurationOutput(ctx)
	utils.RespondWithJson(response, result)
}

const GetLambdaFunctionRegex = `^/2015-03-31/functions/[A-Za-z0-9_-]+$`

func GetLambdaFunction(response http.ResponseWriter, request *http.Request) {
	name := getFunctionName(request.URL.Path)

	log.Info("Getting Lambda Function %s", name)

	ctx := request.Context()
	cfg := settings.FromContext(ctx)
	db := database.CreateConnection(cfg)
	defer db.Close()

	function, err := queries.LatestFunctionByName(ctx, db, name)
	if err != nil {
		msg := log.Error("Unable to get Lambda Function %s: %v", name, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	layers, err := queries.GetLayersForFunction(ctx, db, function)
	if err != nil {
		msg := log.Error("Unable to load Layers for Function %s: %v", name, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	function.Layers = layers
	result := function.ToGetFunctionOutput(ctx)

	utils.RespondWithJson(response, result)
}

const GetFunctionVersionsRegex = `^/2015-03-31/functions/[A-Za-z0-9_-]+/versions$`

func GetFunctionVersions(response http.ResponseWriter, request *http.Request) {
	name := getFunctionName(request.URL.Path)

	log.Info("Getting Versions for Lambda Function %s", name)

	ctx := request.Context()
	cfg := settings.FromContext(ctx)
	db := database.CreateConnection(cfg)
	defer db.Close()

	functions, err := queries.FunctionVersionsByName(ctx, db, name)

	if err != nil {
		msg := log.Error("Unable to get versions for Lambda Function %s: %v", name, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	configs := make([]aws.FunctionConfiguration, len(functions))
	for i, function := range functions {
		configs[i] = *function.ToFunctionConfiguration(ctx)
	}

	results := lambda.ListVersionsByFunctionOutput{
		NextMarker:     nil,
		Versions:       configs,
		ResultMetadata: middleware.Metadata{},
	}

	utils.RespondWithJson(response, results)
}

const GetFunctionCodeSigningRegex = `/2020-06-30/functions/[0-9A-Za-z_-]+/code-signing-config`

func GetFunctionCodeSigning(response http.ResponseWriter, request *http.Request) {
	result := lambda.GetFunctionCodeSigningConfigOutput{}
	utils.RespondWithJson(response, result)
}

const InvokeFunctionRegex = `/2015-03-31/functions/[0-9A-Za-z_-]+/invocations`

func InvokeFunction(response http.ResponseWriter, request *http.Request) {
	name := getFunctionName(request.URL.Path)
	manager.Invoke(name, &response, request)
}

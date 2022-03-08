package lambda

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"io/ioutil"
	"myaws/database"
	"myaws/lambda/queries"
	"myaws/lambda/types"
	"myaws/log"
	"myaws/utils"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func getLayerName(path string) string {
	parts := strings.Split(path, "/")
	return parts[3]
}

const GetAllLayerVersionsRegex = `^/2018-10-31/layers/[A-Za-z0-9_-]+/versions$`

func GetAllLayerVersions(response http.ResponseWriter, request *http.Request) {
	layerName := getLayerName(request.URL.Path)

	ctx := request.Context()
	db := database.CreateConnection()
	defer db.Close()

	layers, err := queries.LayerByName(ctx, db, layerName)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

	result := lambda.ListLayerVersionsOutput{
		LayerVersions: layersToAwsLayers(layers, ctx),
		NextMarker:    nil,
	}

	utils.RespondWithJson(response, result)
}

func getLayerNameAndVersion(path string) (string, int) {
	parts := strings.Split(path, "/")
	version, _ := strconv.Atoi(parts[5])
	return parts[3], version
}

const GetLayerVersionsRegex = `^/2018-10-31/layers/[A-Za-z0-9_-]+/versions/\d+$`

func GetLayerVersion(response http.ResponseWriter, request *http.Request) {
	layerName, version := getLayerNameAndVersion(request.URL.Path)

	ctx := request.Context()
	db := database.CreateConnection()
	defer db.Close()

	layer, err := queries.LayerByNameAndVersion(ctx, db, layerName, version)
	if err != nil {
		log.Error(err.Error())
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

	log.Info("... found %+v", layer)

	result := lambda.GetLayerVersionOutput{
		CompatibleArchitectures: []aws.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		Content: &aws.LayerVersionContentOutput{
			CodeSize:   layer.CodeSize,
			CodeSha256: &layer.CodeSha256,
		},
		CreatedDate:     &layer.CreatedOn,
		Description:     &layer.Description,
		LayerArn:        layer.GetArn(ctx),
		LayerVersionArn: layer.GetVersionArn(ctx),
		LicenseInfo:     nil,
		Version:         int64(layer.Version),
	}

	utils.RespondWithJson(response, result)
}

func layersToAwsLayers(layers []types.LambdaLayer, ctx context.Context) []aws.LayerVersionsListItem {
	results := make([]aws.LayerVersionsListItem, len(layers))
	for i, layer := range layers {
		results[i] = layer.ToLayerVersionsListItem(ctx)
	}

	return results
}

const PostLayerVersionsRegex = `^/2018-10-31/layers/[A-Za-z0-9_-]+/versions$`

func PostLayerVersions(response http.ResponseWriter, request *http.Request) {
	layerName := getLayerName(request.URL.Path)
	dec := json.NewDecoder(request.Body)

	var body lambda.PublishLayerVersionInput
	err := dec.Decode(&body)

	if err != nil {
		msg := fmt.Sprintf("problem parsing request for Lambda layer %s: %v", layerName, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	log.Info("Layer description: %s", *body.Description)
	log.Info("Layer runtimes: %v", body.CompatibleRuntimes)

	ctx := request.Context()
	db := database.CreateConnection()
	defer db.Close()

	dbRuntimes, err := queries.RuntimeIDsByNames(ctx, db, body.CompatibleRuntimes)
	switch {
	case err == sql.ErrNoRows:
		msg := fmt.Sprintf("unable to find all expected runtimes: %v", body.CompatibleRuntimes)
		http.Error(response, msg, http.StatusNotFound)
	case err != nil:
		msg := fmt.Sprintf("error when querying runtime: %v", err)
		http.Error(response, msg, http.StatusInternalServerError)
	}

	version, err := queries.LatestLayerByName(ctx, db, layerName)
	switch {
	case err == sql.ErrNoRows:
		version = -1
	case err != nil:
		msg := fmt.Sprintf("error when listing versions for %s: %v", layerName, err)
		log.Error(msg)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	log.Info("Found latest verion for layer %s: %v", layerName, version)
	log.Info("Saving %d bytes from zipfile", len(body.Content.ZipFile))

	rawHash := sha256.Sum256(body.Content.ZipFile)
	hash := base64.StdEncoding.EncodeToString(rawHash[:])

	layer := types.LambdaLayer{
		Name:               layerName,
		Version:            version + 1,
		Description:        *body.Description,
		CompatibleRuntimes: body.CompatibleRuntimes,
		CodeSize:           int64(len(body.Content.ZipFile)),
		CodeSha256:         hash,
	}

	destPath := layer.GetDestPath(ctx)
	log.Info("Saving layer %s to %s...", layerName, destPath)
	err = utils.CreateDirs(filepath.Dir(destPath))
	if err != nil {
		msg := log.Error("Unable to create parent directory for layer %s: %v", destPath, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	err = ioutil.WriteFile(destPath, body.Content.ZipFile, 0644)
	if err != nil {
		msg := log.Error("error when saving layer %s: %v", layerName, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	savedLayer, err := queries.InsertLayer(ctx, db, layer, &dbRuntimes)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	result := savedLayer.ToPublishLayerVersionOutput(ctx)

	utils.RespondWithJson(response, result)
}

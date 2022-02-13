package lambda

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"log"
	"myaws/utils"
	"net/http"
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
	db := createConnection(ctx)
	defer db.Close()

	layers, err := getAllLayerVersions(ctx, db, layerName)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

	result := lambda.ListLayerVersionsOutput{
		LayerVersions: layersToAwsLayers(layers),
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
	db := createConnection(ctx)
	defer db.Close()

	layer, err := getLayerVersion(ctx, db, layerName, version)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

	result := lambda.GetLayerVersionOutput{
		CompatibleArchitectures: []types.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		Content: &types.LayerVersionContentOutput{
			CodeSize:   layer.CodeSize,
			CodeSha256: &layer.CodeSha256,
		},
		CreatedDate:     &layer.CreatedOn,
		Description:     &layer.Description,
		LayerArn:        layer.getArn(),
		LayerVersionArn: layer.getVersionArn(),
		LicenseInfo:     nil,
		Version:         int64(layer.Version),
	}

	utils.RespondWithJson(response, result)
}

func layersToAwsLayers(layers []LambdaLayer) []types.LayerVersionsListItem {
	results := make([]types.LayerVersionsListItem, len(layers))
	for i, layer := range layers {
		results[i] = layer.toLayerVersionsListItem()
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

	log.Printf("Layer description: %s", *body.Description)
	log.Printf("Layer runtimes: %v", body.CompatibleRuntimes)

	ctx := request.Context()
	db := createConnection(ctx)
	defer db.Close()

	version, err := getLatestLayerVersion(ctx, db, layerName)
	switch {
	case err == sql.ErrNoRows:
		version = -1
	case err != nil:
		msg := fmt.Sprintf("error when listing versions for %s: %v", layerName, err)
		log.Print(msg)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	log.Printf("Found latest verion for layer %s: %v", layerName, version)
	log.Printf("Decompressing %d bytes from zipfile", len(body.Content.ZipFile))

	rawHash := sha256.Sum256(body.Content.ZipFile)
	hash := base64.StdEncoding.EncodeToString(rawHash[:])

	layer := LambdaLayer{
		Name:               layerName,
		Version:            version + 1,
		Description:        *body.Description,
		CompatibleRuntimes: body.CompatibleRuntimes,
		CodeSize:           int64(len(body.Content.ZipFile)),
		CodeSha256:         hash,
	}

	destPath := layer.getDestPath()
	log.Printf("Saving layer %s to %s...", layerName, destPath)

	err = utils.DecompressZipFile(body.Content.ZipFile, destPath)
	if err != nil {
		msg := fmt.Sprintf("error when saving layer %s: %v", layerName, err)
		http.Error(response, msg, http.StatusInternalServerError)
		return
	}

	savedLayer, err := addLayer(ctx, db, layer)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	result := savedLayer.toPublishLayerVersionOutput()

	utils.RespondWithJson(response, result)
}

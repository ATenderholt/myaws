package lambda

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"log"
	"myaws/utils"
	"net/http"
	"strings"
)

func LayerHandler(response http.ResponseWriter, request *http.Request) {
	log.Printf("--- Request %s %q ---", request.Method, request.URL.Path)

	parts := strings.Split(request.URL.Path, "/")
	layerName := parts[3]

	log.Printf("Processing for Lambda layer name '%s'", layerName)
	var err error
	switch request.Method {
	case "GET":
		err = handleLayerGet(&layerName, &response, request)
	case "POST":
		err = handleLayerPost(&layerName, &response, request)
	}

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

}

func handleLayerGet(layerName *string, response *http.ResponseWriter, request *http.Request) error {
	log.Printf("Querying for versions of layer %s", *layerName)

	ctx := request.Context()
	db := createConnection(ctx)
	defer db.Close()

	layers, err := getAllLayerVersions(ctx, db, *layerName)
	if err != nil {
		return err
	}

	result := lambda.ListLayerVersionsOutput{
		LayerVersions: layersToAwsLayers(layers),
		NextMarker:    nil,
	}

	err = json.NewEncoder(*response).Encode(result)
	if err != nil {
		return fmt.Errorf("unable to return mashalled response for %+v: %v", result, err)
	}

	return nil
}

func layersToAwsLayers(layers []LambdaLayer) []types.LayerVersionsListItem {
	results := make([]types.LayerVersionsListItem, len(layers))
	for i, layer := range layers {
		result := types.LayerVersionsListItem{
			CompatibleArchitectures: nil,
			CompatibleRuntimes:      layer.CompatibleRuntimes,
			CreatedDate:             &layer.CreatedOn,
			Description:             &layer.Description,
			LayerVersionArn:         layer.getVersionArn(),
			LicenseInfo:             nil,
			Version:                 int64(layer.Version),
		}

		results[i] = result
	}

	return results
}

func handleLayerPost(layerName *string, response *http.ResponseWriter, request *http.Request) error {
	dec := json.NewDecoder(request.Body)

	var body lambda.PublishLayerVersionInput
	err := dec.Decode(&body)

	if err != nil {
		return fmt.Errorf("problem parsing request for Lambda layer %s: %v", *layerName, err)
	}

	log.Printf("Layer description: %s", *body.Description)
	log.Printf("Layer runtimes: %v", body.CompatibleRuntimes)

	ctx := request.Context()
	db := createConnection(ctx)
	defer db.Close()

	version, err := getLatestLayerVersion(ctx, db, *layerName)
	switch {
	case err == sql.ErrNoRows:
		version = -1
	case err != nil:
		return fmt.Errorf("error when listing versions for %s: %v", *layerName, err)
	}

	log.Printf("Found latest verion for layer %s: %v", *layerName, version)
	log.Printf("Decompressing %d bytes from zipfile", len(body.Content.ZipFile))

	layer := LambdaLayer{
		Name:               *layerName,
		Version:            version + 1,
		Description:        *body.Description,
		CompatibleRuntimes: body.CompatibleRuntimes,
	}

	destPath := layer.getDestPath()
	log.Printf("Saving layer %s to %s...", *layerName, destPath)

	err = utils.DecompressZipFile(body.Content.ZipFile, destPath)
	if err != nil {
		return fmt.Errorf("error when saving layer %s: %v", *layerName, err)
	}

	savedLayer, err := addLayer(ctx, db, layer)
	if err != nil {
		return err
	}

	rawHash := sha256.Sum256(body.Content.ZipFile)
	hash := fmt.Sprintf("%x", rawHash)
	content := types.LayerVersionContentOutput{
		CodeSha256: &hash,
		CodeSize:   int64(len(body.Content.ZipFile)),
	}

	result := lambda.PublishLayerVersionOutput{
		CompatibleArchitectures: nil,
		CompatibleRuntimes:      savedLayer.CompatibleRuntimes,
		Content:                 &content,
		CreatedDate:             &savedLayer.CreatedOn,
		Description:             &savedLayer.Description,
		LayerArn:                savedLayer.getArn(),
		LayerVersionArn:         savedLayer.getVersionArn(),
		LicenseInfo:             nil,
		Version:                 int64(savedLayer.Version),
	}

	err = json.NewEncoder(*response).Encode(result)
	if err != nil {
		return fmt.Errorf("unable to return mashalled response for %+v: %v", result, err)
	}

	return nil
}

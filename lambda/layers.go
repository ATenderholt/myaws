package lambda

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"myaws/config"
	"myaws/utils"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func LayerHandler(response http.ResponseWriter, request *http.Request) {
	log.Printf("--- Request %s %q ---", request.Method, request.URL.Path)

	parts := strings.Split(request.URL.Path, "/")
	layerName := parts[3]

	log.Printf("Processing for Lambda layer name '%s'", layerName)
	if request.Method == "POST" {
		err := handleLayerPost(&layerName, &response, request)
		if err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
		}
	}

}

func handleLayerPost(layerName *string, response *http.ResponseWriter, request *http.Request) error {
	dec := json.NewDecoder(request.Body)

	var body PublishLayerVersionBody
	err := dec.Decode(&body)

	if err != nil {
		return fmt.Errorf("problem parsing request for Lambda layer %s: %v", *layerName, err)
	}

	log.Printf("Layer description: %s", *body.Description)
	log.Printf("Layer runtimes: %v", body.CompatibleRuntimes)

	versions, err := listLayerVersions(layerName)
	if err != nil {
		return fmt.Errorf("error when listing versions for %s: %v", *layerName, err)
	}

	log.Printf("Found following verions for layer %s: %v", *layerName, versions)
	log.Printf("Decompressing %d bytes from zipfile", len(body.Content.ZipFile))

	destPath := filepath.Join(config.GetSettings().GetDataPath(), "lambda", "layers", *layerName,
		strconv.Itoa(len(versions)), "content")

	err = utils.DecompressZipFile(body.Content.ZipFile, destPath)
	if err != nil {
		return fmt.Errorf("error when saving layer %s: %v", *layerName, err)
	}

	ctx := request.Context()
	db := createConnection(ctx)
	defer db.Close()

	return nil
}

func listLayerVersions(layerName *string) ([]int, error) {
	settings := config.GetSettings()
	layerPath := filepath.Join(settings.GetDataPath(), "lambda", "layers", *layerName)
	log.Printf("Making directory %s if it does not exist", layerPath)

	err := os.MkdirAll(layerPath, 0700)
	if err != nil {
		log.Printf("Unable to make directory %s", layerPath)
		return nil, err
	}

	log.Printf("Listing layer versions in %s", layerPath)
	contents, err := ioutil.ReadDir(layerPath)
	if err != nil {
		log.Printf("Unable to list contents of directory %s", layerPath)
		return nil, err
	}

	versions := make([]int, len(contents))
	for i, value := range contents {
		name := value.Name()
		intValue, err := strconv.Atoi(name)
		if err != nil {
			log.Printf("Error when converting %s to number in %s", name, layerPath)
			return nil, err
		}

		versions[i] = intValue
	}

	sort.Ints(versions)
	return versions, nil
}

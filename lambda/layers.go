package lambda

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"myaws/config"
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

	content := ZipContent{Content: body.Content.ZipFile, Length: int64(len(body.Content.ZipFile))}
	reader, err := zip.NewReader(content, content.Length)
	if err != nil {
		return fmt.Errorf("error when reading zip: %v", err)
	}

	for _, f := range reader.File {
		filePath := filepath.Join(config.GetSettings().GetDataPath(), "lambda", "layers", *layerName,
			strconv.Itoa(len(versions)), "content", f.Name)

		if f.FileInfo().IsDir() {
			log.Printf("Creating directory %s", filePath)
			err := os.MkdirAll(filePath, 0755)
			if err != nil {
				return fmt.Errorf("unable to create %s: %v", filePath, err)
			}
			continue
		}

		log.Printf("Saving %s ...", filePath)

		dirPath := filepath.Dir(filePath)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			return fmt.Errorf("unable to create diretory %s: %v", dirPath, err)
		}

		destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("unable to create file %s: %v", filePath, err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			destFile.Close()
			return fmt.Errorf("unable to open file in zip %s: %v", fileInArchive, err)
		}

		_, err = io.Copy(destFile, fileInArchive)
		if err != nil {
			destFile.Close()
			fileInArchive.Close()
			return fmt.Errorf("problem saving file %s: %v", filePath, err)
		}

		destFile.Close()
		fileInArchive.Close()
	}

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

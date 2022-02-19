package main

import (
	"github.com/docker/docker/api/types/mount"
	"io"
	"myaws/config"
	"myaws/database"
	"myaws/docker"
	"myaws/lambda"
	"myaws/log"
	"net/http"
	"path/filepath"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
)

const imageS3 = "bitnami/minio:2022.2.16"

type route struct {
	pattern *regexp.Regexp
	method  string
	handler http.Handler
}

type RegexHandler struct {
	routes []*route
}

func (h *RegexHandler) Handler(pattern string, method string, handler http.Handler) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
	}
	h.routes = append(h.routes, &route{regex, method, handler})
}

func (h *RegexHandler) HandleFunc(pattern string, method string, handler func(http.ResponseWriter, *http.Request)) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
	}
	h.routes = append(h.routes, &route{regex, method, http.HandlerFunc(handler)})
}

func (h *RegexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Info("--- Request %s %q ---", r.Method, r.URL.Path)
	log.Info("Query:")
	for key, value := range r.URL.Query() {
		log.Info("    %s = %s", key, value)
	}

	log.Info("Headers:")
	for key, value := range r.Header {
		log.Info("   %s : %s", key, value)
	}

	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) && route.method == r.Method {
			route.handler.ServeHTTP(w, r)
			return
		}
	}

	log.Info(" ---- %s %s NOT HANDLED BY REGEX --- ", r.Method, r.URL.Path)
	body, _ := io.ReadAll(r.Body)
	log.Info("Body: %s", body)

	//url := fmt.Sprintf("%s://%s%s", "http", "localhost:9324", r.RequestURI)

	//proxyReq, _ := http.NewRequest(r.Method, url, bytes.NewReader(body))
	//proxyReq.Header = r.Header

	//client := &http.Client{}
	//resp, err := client.Do(proxyReq)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusBadGateway)
	//	return
	//}

	http.NotFound(w, r)

	//defer resp.Body.Close()
	//w.WriteHeader(resp.StatusCode)
	//io.Copy(w, resp.Body)
}

func main() {
	settings := config.GetSettings()
	log.Info("Settings: %+v", *settings)

	initializeDb()
	initializeDocker()

	handler := RegexHandler{}
	handler.HandleFunc(lambda.GetAllLayerVersionsRegex, http.MethodGet, lambda.GetAllLayerVersions)
	handler.HandleFunc(lambda.GetLayerVersionsRegex, http.MethodGet, lambda.GetLayerVersion)
	handler.HandleFunc(lambda.PostLayerVersionsRegex, http.MethodPost, lambda.PostLayerVersions)
	handler.HandleFunc(lambda.GetLambdaFunctionRegex, http.MethodGet, lambda.GetLambdaFunction)
	handler.HandleFunc(lambda.GetFunctionCodeSigningRegex, http.MethodGet, lambda.GetFunctionCodeSigning)
	handler.HandleFunc(lambda.GetFunctionVersionsRegex, http.MethodGet, lambda.GetFunctionVersions)
	handler.HandleFunc(lambda.PostLambdaFunctionRegex, http.MethodPost, lambda.PostLambdaFunction)

	http.Handle("/", &handler)

	log.Panic(http.ListenAndServe(":8080", nil).Error())
}

func initializeDb() {
	var migrations database.Migrations
	migrations.AddAll(lambda.Migrations)

	log.Info("Initializing DB with %d Migrations.", migrations.Size())
	database.Initialize(migrations)
}

func initializeDocker() {
	client := docker.NewController()
	client.EnsureImage(imageS3)

	minio := docker.Container{
		Name:  "s3",
		Image: imageS3,
		Mounts: []mount.Mount{
			{
				Source: filepath.Join(config.GetSettings().GetDataPath()),
				Target: "/data",
				Type:   mount.TypeBind,
			},
		},
	}

	err := client.Start(minio)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"context"
	"errors"
	"io"
	"myaws/config"
	"myaws/database"
	"myaws/docker"
	"myaws/lambda"
	"myaws/log"
	"myaws/s3"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s := <-c
		log.Info("Received signal %v", s)
		cancel()
	}()

	if err := start(ctx); err != nil {
		log.Error("Failed to start: %v", err)
	}
}

func start(ctx context.Context) error {
	log.Info("Starting up ...")
	settings := config.GetSettings()
	log.Info("Settings: %+v", *settings)

	initializeDb()
	initializeDocker()
	srv, err := serveHTTP()
	if err != nil {
		return err
	}

	<-ctx.Done()

	log.Info("Shutting down ...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer func() {
		cancel()
	}()

	err = srv.Shutdown(ctxShutDown)
	if err != nil {
		log.Error("Error when shutting down HTTP server")
	}

	err = docker.ShutdownAll()
	if err != nil {
		log.Error("Errors when shutting down docker containers: %v", err)
	}

	return nil
}

func initializeDb() {
	var migrations database.Migrations
	migrations.AddAll(lambda.Migrations)

	log.Info("Initializing DB with %d Migrations.", migrations.Size())
	database.Initialize(migrations)
}

func initializeDocker() {
	docker.EnsureImage(s3.Image)
	err := docker.Start(s3.Container)
	if err != nil {
		panic(err)
	}
}

func serveHTTP() (srv *http.Server, err error) {
	mux := http.NewServeMux()

	handler := RegexHandler{}
	handler.HandleFunc(lambda.GetAllLayerVersionsRegex, http.MethodGet, lambda.GetAllLayerVersions)
	handler.HandleFunc(lambda.GetLayerVersionsRegex, http.MethodGet, lambda.GetLayerVersion)
	handler.HandleFunc(lambda.PostLayerVersionsRegex, http.MethodPost, lambda.PostLayerVersions)
	handler.HandleFunc(lambda.GetLambdaFunctionRegex, http.MethodGet, lambda.GetLambdaFunction)
	handler.HandleFunc(lambda.GetFunctionCodeSigningRegex, http.MethodGet, lambda.GetFunctionCodeSigning)
	handler.HandleFunc(lambda.GetFunctionVersionsRegex, http.MethodGet, lambda.GetFunctionVersions)
	handler.HandleFunc(lambda.PostLambdaFunctionRegex, http.MethodPost, lambda.PostLambdaFunction)
	handler.HandleFunc("/", http.MethodHead, s3.ProxyToMinio)
	handler.HandleFunc("/", http.MethodGet, s3.ProxyToMinio)
	handler.HandleFunc("/", http.MethodPut, s3.ProxyToMinio)

	mux.Handle("/", &handler)
	port := 8080

	srv = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		e := srv.ListenAndServe()
		if e != nil && e != http.ErrServerClosed {
			msg := log.Error("Problem starting HTTP server: %v", e)
			err = errors.New(msg)
		}
	}()

	log.Info("Finished starting HTTP server on port %d", port)
	return
}

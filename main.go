package main

import (
	"io"
	"myaws/config"
	"myaws/database"
	"myaws/lambda"
	"myaws/log"
	"net/http"
	"regexp"

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
	settings := config.GetSettings()
	log.Info("Settings: %+v", *settings)

	initializeDb()

	handler := RegexHandler{}
	handler.HandleFunc(lambda.GetAllLayerVersionsRegex, http.MethodGet, lambda.GetAllLayerVersions)
	handler.HandleFunc(lambda.GetLayerVersionsRegex, http.MethodGet, lambda.GetLayerVersion)
	handler.HandleFunc(lambda.PostLayerVersionsRegex, http.MethodPost, lambda.PostLayerVersions)
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

package http

import (
	"io"
	"myaws/log"
	"net/http"
	"regexp"
)

type route struct {
	pattern *regexp.Regexp
	method  string
	handler http.Handler
}

type RegexHandler struct {
	routes []*route
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

	http.NotFound(w, r)
}

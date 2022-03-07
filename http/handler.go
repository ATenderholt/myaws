package http

import (
	"io"
	"myaws/log"
	"myaws/settings"
	"net/http"
	"regexp"
)

var authRegex *regexp.Regexp

func init() {
	temp, err := regexp.Compile(`Credential=(\w+)/\d{8}/([a-z0-9-]+)/(\w+)/aws4_request`)
	if err != nil {
		panic(err)
	}

	authRegex = temp
}

type route struct {
	pattern *regexp.Regexp
	service *string
	method  string
	handler http.Handler
}

type RegexHandler struct {
	config        *settings.Config
	regexRoutes   []*route
	serviceRoutes []*route
}

func (h *RegexHandler) HandleRegex(pattern string, method string, handler func(http.ResponseWriter, *http.Request)) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
	}
	h.regexRoutes = append(h.regexRoutes, &route{regex, nil, method, http.HandlerFunc(handler)})
}

func (h *RegexHandler) HandleAuthHeader(service string, method string, handler func(http.ResponseWriter, *http.Request)) {
	h.serviceRoutes = append(h.serviceRoutes, &route{nil, &service, method, http.HandlerFunc(handler)})
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

	ctx := h.config.NewContext(r.Context())
	r = r.Clone(ctx)

	// Handle regex based Routes first
	for _, route := range h.regexRoutes {
		if route.pattern.MatchString(r.URL.Path) && route.method == r.Method {
			route.handler.ServeHTTP(w, r)
			return
		}
	}

	auth := r.Header.Get("Authorization")
	groups := authRegex.FindStringSubmatch(auth)
	var service string
	if groups == nil {
		log.Error("Unable to match Authorization header: %s", auth)
		service = ""
	} else {
		service = groups[3]
	}

	log.Info("")
	for _, route := range h.serviceRoutes {
		if *route.service == service && route.method == r.Method {
			route.handler.ServeHTTP(w, r)
			return
		}
	}

	log.Info(" ---- %s %s NOT HANDLED BY REGEX OR SERVICE [%s] --- ", r.Method, r.URL.Path, service)

	body, _ := io.ReadAll(r.Body)
	log.Info("Body: %s", body)

	http.NotFound(w, r)
}

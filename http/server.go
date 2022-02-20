package http

import (
	"errors"
	"myaws/lambda"
	"myaws/log"
	"myaws/s3"
	"net/http"
)

func Serve() (srv *http.Server, err error) {
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

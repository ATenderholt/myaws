package http

import (
	"errors"
	"myaws/iam"
	"myaws/lambda"
	"myaws/log"
	"myaws/s3"
	"myaws/sqs"
	"myaws/ssm"
	"net/http"
)

func Serve() (srv *http.Server, err error) {
	mux := http.NewServeMux()

	handler := RegexHandler{}

	handler.HandleAuthHeader("iam", http.MethodPost, iam.Handler)

	handler.HandleRegex(lambda.GetAllLayerVersionsRegex, http.MethodGet, lambda.GetAllLayerVersions)
	handler.HandleRegex(lambda.GetLayerVersionsRegex, http.MethodGet, lambda.GetLayerVersion)
	handler.HandleRegex(lambda.PostLayerVersionsRegex, http.MethodPost, lambda.PostLayerVersions)
	handler.HandleRegex(lambda.GetLambdaFunctionRegex, http.MethodGet, lambda.GetLambdaFunction)
	handler.HandleRegex(lambda.GetFunctionCodeSigningRegex, http.MethodGet, lambda.GetFunctionCodeSigning)
	handler.HandleRegex(lambda.GetFunctionVersionsRegex, http.MethodGet, lambda.GetFunctionVersions)
	handler.HandleRegex(lambda.PostLambdaFunctionRegex, http.MethodPost, lambda.PostLambdaFunction)
	handler.HandleRegex(lambda.PutLambdaConfigurationRegex, http.MethodPut, lambda.PutLambdaConfiguration)
	handler.HandleRegex(lambda.InvokeFunctionRegex, http.MethodPost, lambda.InvokeFunction)

	handler.HandleAuthHeader("s3", http.MethodHead, s3.ProxyToMinio)
	handler.HandleAuthHeader("s3", http.MethodGet, s3.ProxyToMinio)
	handler.HandleAuthHeader("s3", http.MethodPut, s3.ProxyToMinio)
	handler.HandleAuthHeader("s3", http.MethodDelete, s3.ProxyToMinio)

	handler.HandleAuthHeader("ssm", http.MethodPost, ssm.Handler)

	handler.HandleAuthHeader("sqs", http.MethodPost, sqs.ProxyToElasticMQ)

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

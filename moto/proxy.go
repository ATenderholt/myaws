package moto

import (
	"context"
	"errors"
	"io"
	"myaws/database"
	"myaws/log"
	"myaws/moto/queries"
	"myaws/moto/types"
	"myaws/settings"
	"net/http"
	"strings"
)

const (
	Authorization = "Authorization"
	ContentType   = "Content-Type"
	AmzTarget     = "X-Amz-Target"
)

type ShouldPersist func(http.Header, string) bool

func ProxyToMoto(writer *http.ResponseWriter, request *http.Request, service string, shouldPersist ShouldPersist) (in string, out string, err error) {
	log.Info("Proxying %s request to moto ...", service)

	cfg := settings.FromContext(request.Context())
	url := cfg.Moto.BuildUrl(request.URL.Path)

	var payloadBuilder strings.Builder
	requestBody := io.TeeReader(request.Body, &payloadBuilder)
	authorization := request.Header.Get(Authorization)
	contentType := request.Header.Get(ContentType)
	target := request.Header.Get(AmzTarget)

	proxyReq, _ := http.NewRequest(request.Method, url, requestBody)
	proxyReq.Header.Set(ContentType, contentType)
	proxyReq.Header.Set(Authorization, authorization)
	if len(target) > 0 {
		proxyReq.Header.Set(AmzTarget, target)
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	payload := payloadBuilder.String()
	if err != nil {
		msg := log.Error("... unable to proxy to Moto: %v", err)
		return payload, "", errors.New(msg)
	}

	log.Info("Got following response from Moto: %+v", resp)

	for key, value := range resp.Header {
		for _, v := range value {
			(*writer).Header().Add(key, v)
		}

	}

	if shouldPersist(request.Header, payload) {
		err = InsertRequest(service, request, payload)
		if err != nil {
			msg := log.Error("Unable to insert %s request: %v", service, err)
			responseBody, _ := io.ReadAll(resp.Body)
			return payload, string(responseBody), errors.New(msg)
		}
	}

	(*writer).WriteHeader(resp.StatusCode)

	var responseBuilder strings.Builder
	responseBody := io.TeeReader(resp.Body, &responseBuilder)
	io.Copy(*writer, responseBody)
	resp.Body.Close()

	return payload, responseBuilder.String(), nil
}

func InsertRequest(service string, request *http.Request, payload string) error {
	ctx := request.Context()
	cfg := settings.FromContext(ctx)

	db := database.CreateConnection(cfg)
	defer db.Close()

	authorization := request.Header.Get(Authorization)
	contentType := request.Header.Get(ContentType)
	target := request.Header.Get(AmzTarget)

	apiRequest := types.ApiRequest{
		Service:       service,
		Method:        request.Method,
		Path:          request.URL.Path,
		Authorization: authorization,
		Target:        target,
		ContentType:   contentType,
		Payload:       payload,
	}

	err := queries.InsertRequest(ctx, db, &apiRequest)
	if err != nil {
		msg := log.Error("Unable to insert request for %s: %v", apiRequest.Service, err)
		return errors.New(msg)
	}

	return nil
}

func ReplayToMoto(cfg *settings.Config, request types.ApiRequest) error {
	log.Info("Replaying %s request #%d to moto ...", request.Service, request.ID)

	url := cfg.Moto.BuildUrl(request.Path)

	proxyReq, _ := http.NewRequest(request.Method, url, strings.NewReader(request.Payload))
	proxyReq.Header.Set("Content-Type", request.ContentType)
	proxyReq.Header.Set("Authorization", request.Authorization)
	if len(request.Target) > 0 {
		proxyReq.Header.Set("X-Amz-Target", request.Target)
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := queries.ErrorMessage(&request, err)
		log.Error(msg)
		return errors.New(msg)
	}

	log.Info("Got following response from Moto: %+v", resp)
	return nil
}

func ReplayAllToMoto(ctx context.Context) error {
	log.Info("Replaying all requests to moto ...")

	cfg := settings.FromContext(ctx)
	db := database.CreateConnection(cfg)
	defer db.Close()

	dbCtx, cancel := context.WithCancel(ctx)
	results, done, errs := queries.FindAllRequests(dbCtx, db)
	for {
		select {
		case result := <-results:
			err := ReplayToMoto(cfg, result)
			if err != nil {
				cancel()
				msg := log.Error("Unable to replay to moto requests: %v", err)
				return errors.New(msg)
			}
		case err := <-errs:
			cancel()
			if err != nil {
				msg := log.Error("Unable to replay to requests: %v", err)
				return errors.New(msg)
			}
		case <-done:
			cancel()
			log.Info("Done replaying moto requests.")
			return nil
		}
	}
}

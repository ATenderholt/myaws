package moto

import (
	"context"
	"errors"
	"fmt"
	"io"
	"myaws/config"
	"myaws/database"
	"myaws/log"
	"myaws/moto/queries"
	"myaws/moto/types"
	"net/http"
	"strings"
)

func ProxyToMoto(response *http.ResponseWriter, request *http.Request, service string) (in string, out string, err error) {
	log.Info("Proxying %s request to moto ...", service)

	url := fmt.Sprintf("http://%s:%d%s", config.Moto().Host, config.Moto().Port, request.URL.Path)

	var proxyRequestBody strings.Builder
	requestBody := io.TeeReader(request.Body, &proxyRequestBody)
	authorization := request.Header.Get("Authorization")
	contentType := request.Header.Get("Content-Type")

	proxyReq, _ := http.NewRequest(request.Method, url, requestBody)
	proxyReq.Header.Set("Content-Type", contentType)
	proxyReq.Header.Set("Authorization", authorization)

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := log.Error("Problem proxying to Moto: %v", err)
		return proxyRequestBody.String(), "", errors.New(msg)
	}

	log.Info("Got following response from Moto: %+v", resp)

	for key, value := range resp.Header {
		for _, v := range value {
			(*response).Header().Add(key, v)
		}

	}

	ctx := request.Context()
	db := database.CreateConnection()
	defer db.Close()

	apiRequest := types.ApiRequest{
		Service:       service,
		Method:        request.Method,
		Path:          request.URL.Path,
		Authorization: authorization,
		ContentType:   contentType,
		Payload:       proxyRequestBody.String(),
	}

	err = queries.InsertRequest(ctx, db, &apiRequest)
	if err != nil {
		msg := log.Error("Unable to insert request for %s: %v", apiRequest.Service, err)
		return proxyRequestBody.String(), apiRequest.Payload, errors.New(msg)
	}

	(*response).WriteHeader(resp.StatusCode)

	var responseBody strings.Builder
	body := io.TeeReader(resp.Body, &responseBody)
	io.Copy(*response, body)
	resp.Body.Close()

	return proxyRequestBody.String(), apiRequest.Payload, nil
}

func ReplayToMoto(request types.ApiRequest) error {
	log.Info("Replaying %s request #%d to moto ...", request.Service, request.ID)

	url := fmt.Sprintf("http://%s:%d%s", config.Moto().Host, config.Moto().Port, request.Path)

	proxyReq, _ := http.NewRequest(request.Method, url, strings.NewReader(request.Payload))
	proxyReq.Header.Set("Content-Type", request.ContentType)
	proxyReq.Header.Set("Authorization", request.Authorization)

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

	db := database.CreateConnection()
	defer db.Close()

	dbCtx, cancel := context.WithCancel(ctx)
	results, done, errs := queries.FindAllRequests(dbCtx, db)
	for {
		select {
		case result := <-results:
			err := ReplayToMoto(result)
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

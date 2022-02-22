package moto

import (
	"errors"
	"fmt"
	"io"
	"myaws/config"
	"myaws/log"
	"net/http"
	"strings"
)

func ProxyToMoto(response *http.ResponseWriter, request *http.Request, service string) error {
	log.Info("Proxying %s request to moto ...", service)

	url := fmt.Sprintf("http://%s:%d%s", config.Moto().Host, config.Moto().Port, request.URL.Path)
	proxyReq, _ := http.NewRequest(request.Method, url, request.Body)
	proxyReq.Header.Set("Content-Type", request.Header.Get("Content-Type"))
	//proxyReq.Header.Set("Host", "https://iam.us-west-2.amazonaws.com")
	proxyReq.Header.Set("Authorization", request.Header.Get("Authorization"))

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := log.Error("Problem proxying to Moto: %v", err)
		return errors.New(msg)
	}

	log.Info("Got following response from Moto: %+v", resp)

	for key, value := range resp.Header {
		for _, v := range value {
			(*response).Header().Add(key, v)
		}

	}

	(*response).WriteHeader(resp.StatusCode)
	var stringBuilder strings.Builder
	body := io.TeeReader(resp.Body, &stringBuilder)
	io.Copy(*response, body)
	resp.Body.Close()

	if resp.Header.Get("Content-Type") == "text/html; charset=utf-8" {
		log.Info("Response from Moto: %s", stringBuilder.String())
	}

	return nil
}

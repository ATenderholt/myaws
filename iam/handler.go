package iam

import (
	"myaws/log"
	"myaws/moto"
	"net/http"
)

func Handler(response http.ResponseWriter, request *http.Request) {
	req, resp, err := moto.ProxyToMoto(&response, request, "iam")
	if err == nil {
		return
	}

	if len(req) > 0 {
		log.Error("IAM Request: ", req)
	}

	if len(resp) > 0 {
		log.Error("IAM Response:", resp)
	}

	http.Error(response, err.Error(), http.StatusInternalServerError)
}

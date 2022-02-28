package iam

import (
	"myaws/log"
	"myaws/moto"
	"net/http"
)

func Handler(response http.ResponseWriter, request *http.Request) {
	in, out, err := moto.ProxyToMoto(&response, request, "iam")
	log.Debug("IAM Request Payload: %s", in)
	log.Debug("IAM Response Body: %s", out)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	err = moto.InsertRequest("iam", request, in)
	if err != nil {
		msg := log.Error("Unable to insert IAM request: %v", err)
		http.Error(response, msg, http.StatusInternalServerError)
	}
}

package iam

import (
	"myaws/log"
	"myaws/moto"
	"net/http"
	"strings"
)

var excludes = [...]string{"GetRole", "ListRolePolicies", "ListAttachedRolePolicies"}

func Handler(response http.ResponseWriter, request *http.Request) {
	in, out, err := moto.ProxyToMoto(&response, request, "iam")
	log.Debug("IAM Request Payload: %s", in)
	log.Debug("IAM Response Body: %s", out)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, exclude := range excludes {
		if strings.Contains(in, exclude) {
			return
		}
	}

	err = moto.InsertRequest("iam", request, in)
	if err != nil {
		msg := log.Error("Unable to insert IAM request: %v", err)
		http.Error(response, msg, http.StatusInternalServerError)
	}
}

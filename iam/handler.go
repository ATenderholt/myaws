package iam

import (
	"myaws/log"
	"myaws/moto"
	"net/http"
	"strings"
)

var excludes = [...]string{"GetRole", "ListRolePolicies", "ListAttachedRolePolicies"}

func shouldPersist(_ http.Header, payload string) bool {
	for _, exclude := range excludes {
		if strings.Contains(payload, exclude) {
			return false
		}
	}

	return true
}

func Handler(response http.ResponseWriter, request *http.Request) {
	in, out, err := moto.ProxyToMoto(&response, request, "iam", shouldPersist)
	log.Debug("IAM Request Payload: %s", in)
	log.Debug("IAM Response Body: %s", out)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

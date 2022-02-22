package iam

import (
	"myaws/moto"
	"net/http"
)

func Handler(response http.ResponseWriter, request *http.Request) {
	err := moto.ProxyToMoto(&response, request, "iam")
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

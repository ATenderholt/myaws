package ssm

import (
	"myaws/log"
	"myaws/moto"
	"net/http"
)

var excludes = [...]string{"AmazonSSM.GetParameter", "AmazonSSM.DescribeParameters", "AmazonSSM.ListTagsForResource"}

func shouldPersist(headers http.Header, payload string) bool {
	for _, exclude := range excludes {
		if headers.Get("X-Amz-Target") == exclude {
			return false
		}
	}

	return true
}

func Handler(response http.ResponseWriter, request *http.Request) {
	in, out, err := moto.ProxyToMoto(&response, request, "ssm", shouldPersist)
	log.Debug("SSM Request Payload: %s", in)
	log.Debug("SSM Response Body: %s", out)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

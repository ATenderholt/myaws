package ssm

import (
	"myaws/log"
	"myaws/moto"
	"net/http"
)

var excludes = [...]string{"AmazonSSM.GetParameter", "AmazonSSM.DescribeParameters", "AmazonSSM.ListTagsForResource"}

func Handler(response http.ResponseWriter, request *http.Request) {
	in, out, err := moto.ProxyToMoto(&response, request, "ssm")
	log.Debug("SSM Request Payload: %s", in)
	log.Debug("SSM Response Body: %s", out)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, exclude := range excludes {
		if request.Header.Get("X-Amz-Target") == exclude {
			return
		}
	}

	err = moto.InsertRequest("ssm", request, in)
	if err != nil {
		msg := log.Error("Unable to insert SSM request: %v", err)
		http.Error(response, msg, http.StatusInternalServerError)
	}
}

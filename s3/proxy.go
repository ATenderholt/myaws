package s3

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"io"
	"myaws/config"
	"myaws/log"
	"net/http"
	"time"
)

var settings = config.GetSettings()

func ProxyToMinio(response http.ResponseWriter, request *http.Request) {
	err := proxyToMinio(&response, request, "us-west-2")
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func proxyToMinio(response *http.ResponseWriter, request *http.Request, region string) error {
	url := fmt.Sprintf("http://%s:%d%s", settings.S3().Host, settings.S3().Port, request.URL.Path)
	body, _ := io.ReadAll(request.Body)
	proxyReq, _ := http.NewRequest(request.Method, url, bytes.NewReader(body))

	credentials := aws.Credentials{AccessKeyID: "minio", SecretAccessKey: "miniosecret"}

	signer := v4.NewSigner()
	err := signer.SignHTTP(request.Context(), credentials, proxyReq,
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"s3", region, time.Now())

	if err != nil {
		msg := log.Error("Problem signing request to Minio: %v", err)
		return errors.New(msg)
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := log.Error("Problem proxying to Minio: %v", err)
		return errors.New(msg)
	}

	log.Info("Got following response from Minio: %+v", resp)
	defer resp.Body.Close()

	for key, value := range resp.Header {
		for _, v := range value {
			(*response).Header().Add(key, v)
		}

	}

	(*response).WriteHeader(resp.StatusCode)
	io.Copy(*response, resp.Body)

	return nil
}

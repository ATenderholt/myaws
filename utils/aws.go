package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"io"
	"myaws/log"
	"net/http"
	"time"
)

func ProxyToAws(response *http.ResponseWriter, request *http.Request, service string, region string) error {
	url := fmt.Sprintf("https://%s.%s.amazonaws.com%s", service, region, request.URL.Path)
	body, _ := io.ReadAll(request.Body)
	proxyReq, _ := http.NewRequest(request.Method, url, bytes.NewReader(body))

	ctx := request.Context()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		msg := log.Error("Unable to load AWS credentials: %+v", err)
		return errors.New(msg)
	}

	credentials, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		msg := log.Error("Unable to get credentials: %v", err)
		return errors.New(msg)
	}

	signer := v4.NewSigner()
	err = signer.SignHTTP(request.Context(), credentials, proxyReq,
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		service, region, time.Now())

	if err != nil {
		msg := log.Error("Problem signing request to AWS: %v", err)
		return errors.New(msg)
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		msg := log.Error("Problem proxying to AWS: %v", err)
		return errors.New(msg)
	}

	log.Info("Got following response from AWS: %+v", resp)
	defer resp.Body.Close()
	awsBody, err := io.ReadAll(resp.Body)
	if err != nil {
		msg := log.Error("Problem reading body from AWS: %+v", err)
		return errors.New(msg)
	}

	log.Info("Body from AWS: %s", awsBody)

	(*response).WriteHeader(resp.StatusCode)
	io.Copy(*response, resp.Body)

	return nil
}

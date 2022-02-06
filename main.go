package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"myaws/config"
	"myaws/lambda"
	"net/http"
)

func logHandler(writer http.ResponseWriter, r *http.Request) {
	log.Printf("--- Request %s %q ---", r.Method, r.URL.Path)
	log.Printf("Query:")
	for key, value := range r.URL.Query() {
		log.Printf("    %s = %s", key, value)
	}

	log.Print("Headers:")
	for key, value := range r.Header {
		log.Printf("   %s : %s", key, value)
	}

	body, _ := io.ReadAll(r.Body)
	log.Printf("Body: %s", body)

	url := fmt.Sprintf("%s://%s%s", "http", "localhost:9324", r.RequestURI)

	proxyReq, _ := http.NewRequest(r.Method, url, bytes.NewReader(body))
	proxyReq.Header = r.Header

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()
	writer.WriteHeader(resp.StatusCode)
	io.Copy(writer, resp.Body)
}

func main() {
	settings := config.GetSettings()
	log.Printf("Settings: %+v", *settings)

	http.HandleFunc("/2018-10-31/layers/", lambda.LayerHandler)
	http.HandleFunc("/", logHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

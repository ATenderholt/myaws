package utils

import (
	"encoding/json"
	"fmt"
	"myaws/log"
	"net/http"
)

func RespondWithJson(response http.ResponseWriter, value interface{}) {
	log.Info("Response: %+v", value)

	err := json.NewEncoder(response).Encode(value)
	if err != nil {
		msg := fmt.Sprintf("unable to return mashalled response for %+v: %v", value, err)
		http.Error(response, msg, http.StatusInternalServerError)
	}
}

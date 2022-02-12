package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func RespondWithJson(response http.ResponseWriter, value interface{}) {
	err := json.NewEncoder(response).Encode(value)
	if err != nil {
		msg := fmt.Sprintf("unable to return mashalled response for %+v: %v", value, err)
		http.Error(response, msg, http.StatusInternalServerError)
	}
}

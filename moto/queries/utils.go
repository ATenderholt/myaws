package queries

import (
	"myaws/moto/types"
	"strings"
)

func ErrorMessage(apiRequest *types.ApiRequest, err error) string {
	var builder strings.Builder
	builder.WriteString("unable to insert request for ")
	builder.WriteString(apiRequest.Service)
	builder.WriteString(": " + err.Error())
	builder.WriteString("   Authorization: " + apiRequest.Authorization)
	builder.WriteString("   Payload: " + apiRequest.Payload)
	builder.WriteString("----")
	return builder.String()
}

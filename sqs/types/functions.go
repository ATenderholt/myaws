package types

import (
	"context"
	"net/http"
)

type ExtraWorkFunction func(ctx context.Context, writer *http.ResponseWriter, proxyResponse *http.Response, payload string) (string, string, error)

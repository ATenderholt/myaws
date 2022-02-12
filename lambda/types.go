package lambda

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"path/filepath"
	"strconv"
)

type LambdaLayer struct {
	ID                 int64
	Name               string
	Version            int
	Description        string
	CreatedOn          string
	CompatibleRuntimes []types.Runtime
	CodeSize           int64
	CodeSha256         string
}

func (layer LambdaLayer) getDestPath() string {
	return filepath.Join(settings.GetDataPath(), "lambda", "layers", layer.Name,
		strconv.Itoa(layer.Version), "content")
}

func (layer LambdaLayer) getArn() *string {
	result := "arn:aws:lambda:" + settings.GetArnFragment() + ":layer:" + layer.Name
	return &result
}

func (layer LambdaLayer) getVersionArn() *string {
	arn := layer.getArn()
	result := *arn + ":" + strconv.Itoa(layer.Version)
	return &result
}

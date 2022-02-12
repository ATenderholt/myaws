package lambda

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda"
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

func (layer LambdaLayer) toPublishLayerVersionOutput() *lambda.PublishLayerVersionOutput {
	return &lambda.PublishLayerVersionOutput{
		CompatibleArchitectures: []types.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		Content: &types.LayerVersionContentOutput{
			CodeSize:   layer.CodeSize,
			CodeSha256: &layer.CodeSha256,
		},
		CreatedDate:     &layer.CreatedOn,
		Description:     &layer.Description,
		LayerArn:        layer.getArn(),
		LayerVersionArn: layer.getVersionArn(),
		LicenseInfo:     nil,
		Version:         int64(layer.Version),
	}
}

func (layer LambdaLayer) toLayerVersionsListItem() types.LayerVersionsListItem {
	return types.LayerVersionsListItem{
		CompatibleArchitectures: []types.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		CreatedDate:             &layer.CreatedOn,
		Description:             &layer.Description,
		LayerVersionArn:         layer.getVersionArn(),
		LicenseInfo:             nil,
		Version:                 int64(layer.Version),
	}
}

package types

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"myaws/config"
	"path/filepath"
	"strconv"
)

var settings = config.GetSettings()

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

func (layer LambdaLayer) GetDestPath() string {
	return filepath.Join(settings.GetDataPath(), "lambda", "layers", layer.Name,
		strconv.Itoa(layer.Version), "content")
}

func (layer LambdaLayer) GetArn() *string {
	result := "arn:aws:lambda:" + settings.GetArnFragment() + ":layer:" + layer.Name
	return &result
}

func (layer LambdaLayer) GetVersionArn() *string {
	arn := layer.GetArn()
	result := *arn + ":" + strconv.Itoa(layer.Version)
	return &result
}

func (layer LambdaLayer) ToPublishLayerVersionOutput() *lambda.PublishLayerVersionOutput {
	return &lambda.PublishLayerVersionOutput{
		CompatibleArchitectures: []types.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		Content: &types.LayerVersionContentOutput{
			CodeSize:   layer.CodeSize,
			CodeSha256: &layer.CodeSha256,
		},
		CreatedDate:     &layer.CreatedOn,
		Description:     &layer.Description,
		LayerArn:        layer.GetArn(),
		LayerVersionArn: layer.GetVersionArn(),
		LicenseInfo:     nil,
		Version:         int64(layer.Version),
	}
}

func (layer LambdaLayer) ToLayerVersionsListItem() types.LayerVersionsListItem {
	return types.LayerVersionsListItem{
		CompatibleArchitectures: []types.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		CreatedDate:             &layer.CreatedOn,
		Description:             &layer.Description,
		LayerVersionArn:         layer.GetVersionArn(),
		LicenseInfo:             nil,
		Version:                 int64(layer.Version),
	}
}

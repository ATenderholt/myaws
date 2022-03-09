package types

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"myaws/settings"
	"path/filepath"
	"strconv"
	"strings"
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

func (layer LambdaLayer) GetDestPath(ctx context.Context) string {
	fileName := strconv.Itoa(layer.Version) + ".zip"
	cfg := settings.FromContext(ctx)
	return filepath.Join(cfg.DataPath(), "lambda", "layers", layer.Name, fileName)
}

func (layer LambdaLayer) GetArn(ctx context.Context) *string {
	cfg := settings.FromContext(ctx)
	result := "arn:aws:lambda:" + cfg.Region + ":" + cfg.AccountNumber + ":layer:" + layer.Name
	return &result
}

func (layer LambdaLayer) GetVersionArn(ctx context.Context) *string {
	arn := layer.GetArn(ctx)
	result := *arn + ":" + strconv.Itoa(layer.Version)
	return &result
}

func LayerFromArn(arn string) LambdaLayer {
	parts := strings.Split(arn, ":")
	version, err := strconv.Atoi(parts[7])

	if err != nil {
		panic(err)
	}

	return LambdaLayer{
		Name:    parts[6],
		Version: version,
	}
}

func (layer LambdaLayer) ToPublishLayerVersionOutput(ctx context.Context) *lambda.PublishLayerVersionOutput {
	return &lambda.PublishLayerVersionOutput{
		CompatibleArchitectures: []types.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		Content: &types.LayerVersionContentOutput{
			CodeSize:   layer.CodeSize,
			CodeSha256: &layer.CodeSha256,
		},
		CreatedDate:     &layer.CreatedOn,
		Description:     &layer.Description,
		LayerArn:        layer.GetArn(ctx),
		LayerVersionArn: layer.GetVersionArn(ctx),
		LicenseInfo:     nil,
		Version:         int64(layer.Version),
	}
}

func (layer LambdaLayer) ToLayerVersionsListItem(ctx context.Context) types.LayerVersionsListItem {
	return types.LayerVersionsListItem{
		CompatibleArchitectures: []types.Architecture{},
		CompatibleRuntimes:      layer.CompatibleRuntimes,
		CreatedDate:             &layer.CreatedOn,
		Description:             &layer.Description,
		LayerVersionArn:         layer.GetVersionArn(ctx),
		LicenseInfo:             nil,
		Version:                 int64(layer.Version),
	}
}

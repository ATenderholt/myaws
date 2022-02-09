package lambda

import (
	"myaws/config"
	"path/filepath"
	"strconv"
)

type LayerVersionContentInput struct {
	S3Bucket        *string
	S3Key           *string
	S3ObjectVersion *string
	ZipFile         []byte
}

type Architecture string
type Runtime string

type PublishLayerVersionBody struct {
	Content                 LayerVersionContentInput
	LayerName               *string
	CompatibleArchitectures []Architecture
	CompatibleRuntimes      []Runtime
	Description             *string
	LicenseInfo             *string
}

type LambdaLayer struct {
	ID                 int
	Name               string
	Version            int
	Description        string
	CreatedOn          int64
	CompatibleRuntimes []Runtime
}

func (layer LambdaLayer) getDestPath() string {
	return filepath.Join(config.GetSettings().GetDataPath(), "lambda", "layers", layer.Name,
		strconv.Itoa(layer.Version), "content")
}

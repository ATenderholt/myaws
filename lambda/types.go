package lambda

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"myaws/config"
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
}

func (layer LambdaLayer) getDestPath() string {
	return filepath.Join(config.GetSettings().GetDataPath(), "lambda", "layers", layer.Name,
		strconv.Itoa(layer.Version), "content")
}

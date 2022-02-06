package lambda

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

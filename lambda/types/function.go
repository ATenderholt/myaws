package types

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go/middleware"
	"myaws/utils"
	"path/filepath"
	"time"
)

type Function struct {
	ID            int64
	FunctionName  string
	Description   string
	Handler       string
	Role          string
	DeadLetterArn string
	Layers        []LambdaLayer
	MemorySize    int32
	Runtime       aws.Runtime
	Timeout       int32
	CodeSha256    string
	CodeSize      int64

	Environment *aws.Environment
	Tags        map[string]string

	// For network connectivity to Amazon Web Services resources in a VPC, specify a
	// TODO : VpcConfig *types.VpcConfig

	// The date and time that the function was last updated, in ISO-8601 format
	// (https://www.w3.org/TR/NOTE-datetime) (YYYY-MM-DDThh:mm:ss.sTZD).
	LastModified int64

	// The status of the last update that was performed on the function. This is first
	// set to Successful after function creation completes.
	LastUpdateStatus aws.LastUpdateStatus

	// The reason for the last update that was performed on the function.
	LastUpdateStatusReason *string

	// The reason code for the last update that was performed on the function.
	LastUpdateStatusReasonCode aws.LastUpdateStatusReasonCode

	// The type of deployment package. Set to Image for container image and set Zip for
	// .zip file archive.
	PackageType aws.PackageType

	// The latest updated revision of the function or alias.
	RevisionId *string

	// The current state of the function. When the state is Inactive, you can
	// reactivate the function by invoking it.
	State aws.State

	// The reason for the function's current state.
	StateReason *string

	// The reason code for the function's current state. When the code is Creating, you
	// can't invoke or modify the function.
	StateReasonCode aws.StateReasonCode

	// The version of the Lambda function.
	Version string
}

func CreateFunction(input *lambda.CreateFunctionInput) *Function {
	var deadLetterArn string
	if input.DeadLetterConfig != nil {
		deadLetterArn = *input.DeadLetterConfig.TargetArn
	}

	return &Function{
		FunctionName:  *input.FunctionName,
		Role:          *input.Role,
		Description:   utils.StringOrEmpty(input.Description),
		Handler:       *input.Handler,
		DeadLetterArn: deadLetterArn,
		Layers:        nil, // TODO : body.Layers,
		MemorySize:    utils.Int32OrDefault(input.MemorySize, 128),
		Runtime:       input.Runtime,
		Timeout:       utils.Int32OrDefault(input.Timeout, 3),
		Environment:   EnvironmentOrEmpty(input.Environment),
		Tags:          input.Tags,
		LastModified:  time.Now().UnixMilli(),
	}
}

func (f Function) ToCreateFunctionOutput() *lambda.CreateFunctionOutput {
	lastModified := time.UnixMilli(f.LastModified).Format(timeFormat)

	return &lambda.CreateFunctionOutput{
		Architectures:    nil,
		CodeSha256:       &f.CodeSha256,
		CodeSize:         f.CodeSize,
		DeadLetterConfig: &aws.DeadLetterConfig{TargetArn: &f.DeadLetterArn},
		Description:      &f.Description,
		Environment: &aws.EnvironmentResponse{
			Error:     nil,
			Variables: f.Environment.Variables,
		},
		FileSystemConfigs:          nil,
		FunctionArn:                nil,
		FunctionName:               &f.FunctionName,
		Handler:                    &f.Handler,
		ImageConfigResponse:        nil,
		KMSKeyArn:                  nil,
		LastModified:               &lastModified,
		LastUpdateStatus:           "",
		LastUpdateStatusReason:     nil,
		LastUpdateStatusReasonCode: "",
		Layers:                     layersToAws(f.Layers),
		MasterArn:                  nil,
		MemorySize:                 &f.MemorySize,
		PackageType:                "Zip",
		RevisionId:                 nil,
		Role:                       &f.Role,
		Runtime:                    f.Runtime,
		SigningJobArn:              nil,
		SigningProfileVersionArn:   nil,
		State:                      aws.StateActive,
		StateReason:                nil,
		StateReasonCode:            "",
		Timeout:                    &f.Timeout,
		TracingConfig:              nil,
		Version:                    &f.Version,
		VpcConfig:                  nil,
		ResultMetadata:             middleware.Metadata{},
	}
}

func (f *Function) ToFunctionConfiguration() *aws.FunctionConfiguration {
	lastModified := timeMillisToString(f.LastModified)

	return &aws.FunctionConfiguration{
		Architectures:              nil,
		CodeSha256:                 &f.CodeSha256,
		CodeSize:                   f.CodeSize,
		DeadLetterConfig:           nil,
		Description:                &f.Description,
		Environment:                nil,
		FileSystemConfigs:          nil,
		FunctionArn:                f.GetArn(),
		FunctionName:               &f.FunctionName,
		Handler:                    &f.Handler,
		ImageConfigResponse:        nil,
		KMSKeyArn:                  nil,
		LastModified:               &lastModified,
		LastUpdateStatus:           "",
		LastUpdateStatusReason:     nil,
		LastUpdateStatusReasonCode: "",
		Layers:                     nil,
		MasterArn:                  nil,
		MemorySize:                 &f.MemorySize,
		PackageType:                "Zip",
		RevisionId:                 nil,
		Role:                       &f.Role,
		Runtime:                    f.Runtime,
		SigningJobArn:              nil,
		SigningProfileVersionArn:   nil,
		State:                      "Active",
		StateReason:                nil,
		StateReasonCode:            "",
		Timeout:                    &f.Timeout,
		TracingConfig:              nil,
		Version:                    &f.Version,
		VpcConfig:                  nil,
	}
}

func (f *Function) ToGetFunctionOutput() *lambda.GetFunctionOutput {
	config := f.ToFunctionConfiguration()
	code := aws.FunctionCodeLocation{}
	one := int32(-1)
	concurrency := aws.Concurrency{ReservedConcurrentExecutions: &one}
	return &lambda.GetFunctionOutput{
		Code:           &code,
		Concurrency:    &concurrency,
		Configuration:  config,
		Tags:           nil,
		ResultMetadata: middleware.Metadata{},
	}
}

func layersToAws(layers []LambdaLayer) []aws.Layer {
	results := make([]aws.Layer, len(layers))
	for i, layer := range layers {
		results[i] = aws.Layer{
			Arn:      layer.GetVersionArn(),
			CodeSize: layer.CodeSize,
		}
	}

	return results
}

func (f *Function) GetDestPath() string {
	return filepath.Join(settings.GetDataPath(), "lambda", "functions", f.FunctionName,
		f.Version, "content")
}

func (f *Function) GetArn() *string {
	result := "arn:aws:lambda:" + settings.GetArnFragment() + ":function:" + f.FunctionName
	return &result
}

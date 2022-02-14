package types

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go/middleware"
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
	LastModified string

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
	Version *string
}

func (f Function) ToCreateFunctionOutput() *lambda.CreateFunctionOutput {
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
		LastModified:               &f.LastModified,
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
		Version:                    f.Version,
		VpcConfig:                  nil,
		ResultMetadata:             middleware.Metadata{},
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

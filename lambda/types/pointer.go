package types

import aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"

func EnvironmentOrEmpty(environment *aws.Environment) *aws.Environment {
	if environment != nil {
		return environment
	}

	emptyMap := make(map[string]string, 0)
	return &aws.Environment{Variables: emptyMap}
}

package config

import (
	"flag"
	"os"
	"path/filepath"
)

type HostSettings struct {
	Host string
	Port int
}

type Settings struct {
	accountNumber string
	dataPath      string
	debug         bool
	region        string
	s3            HostSettings
	sqs           HostSettings
}

var instance Settings

func init() {
	instance = Settings{}
	flag.StringVar(&instance.accountNumber, "account-number", "271828182845", "Account number")
	flag.StringVar(&instance.dataPath, "data-path", "data", "Path to data directory")
	flag.BoolVar(&instance.debug, "debug", false, "Enable trace debugging")
	flag.StringVar(&instance.s3.Host, "s3-host", "localhost", "Host for S3 / minio")
	flag.IntVar(&instance.s3.Port, "s3-port", 9000, "Base port for S3 / minio")
	flag.StringVar(&instance.sqs.Host, "sqs-host", "localhost", "Host for SQS / ElasticMQ")
	flag.IntVar(&instance.sqs.Port, "sqs-port", 9324, "Base port for SQS / ElasticMQ")
	flag.Parse()

	instance.region = "us-west-2"
}

func AccountNumber() string {
	return instance.accountNumber
}

func GetDataPath() string {
	if instance.dataPath[0] == '/' {
		return instance.dataPath
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return filepath.Join(cwd, instance.dataPath)
}

func GetArnFragment() string {
	return instance.region + ":" + instance.accountNumber
}

func IsDebug() bool {
	return instance.debug
}

func Region() string {
	return instance.region
}

func S3() HostSettings {
	return instance.s3
}

func SQS() HostSettings {
	return instance.sqs
}

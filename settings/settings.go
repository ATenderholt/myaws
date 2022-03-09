package settings

import (
	"context"
	"os"
	"path/filepath"
)

const (
	configContextKey = contextKey("config")

	DefaultAccountNumber = "271828182845"
	DefaultRegion        = "us-west-2"

	DefaultDataPath = "data"

	DefaultLambdaPort = 9002
	DefaultMotoPort   = 9326
	DefaultS3Port     = 9000
	DefaultSqsPort    = 9324
)

type contextKey string

type Config struct {
	AccountNumber string
	IsDebug       bool
	Region        string

	Database *Database
	Lambda   *Server
	Moto     *Server
	S3       *Server
	SQS      *Server

	dataPath string
}

func (config *Config) ArnFragment() string {
	return config.Region + ":" + config.AccountNumber
}

func (config *Config) DataPath() string {
	if config.dataPath[0] == '/' {
		return config.dataPath
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return filepath.Join(cwd, config.dataPath)
}

func (config *Config) DbConnectionString() string {
	return config.Database.connectionString(config.dataPath)
}

func (config *Config) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, configContextKey, config)
}

func FromContext(ctx context.Context) *Config {
	cfg, ok := ctx.Value(configContextKey).(*Config)
	if ok {
		return cfg
	}

	return DefaultConfig()
}

func DefaultConfig() *Config {
	return &Config{
		AccountNumber: DefaultAccountNumber,
		IsDebug:       false,
		Region:        DefaultRegion,
		Database:      DefaultDatabase(),
		Lambda:        NewLocalhostServer(DefaultLambdaPort),
		Moto:          NewLocalhostServer(DefaultMotoPort),
		S3:            NewLocalhostServer(DefaultS3Port),
		SQS:           NewLocalhostServer(DefaultSqsPort),
		dataPath:      DefaultDataPath,
	}
}

package config

import (
	"flag"
	"sync"
)

type Settings struct {
	accountNumber string
	dataPath      string
	region        string
}

var once sync.Once
var instance Settings

func GetSettings() *Settings {
	once.Do(func() {
		instance = Settings{}
		flag.StringVar(&instance.dataPath, "data-path", "data", "Path to data directory")
		flag.Parse()

		instance.accountNumber = "000000000000"
		instance.region = "us-west-2"
	})

	return &instance
}

func (settings *Settings) GetDataPath() string {
	return settings.dataPath
}

func (settings *Settings) GetArnFragment() string {
	return settings.accountNumber + ":" + settings.region
}

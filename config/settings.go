package config

import (
	"flag"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	accountNumber string
	dataPath      string
	debug         bool
	region        string
}

var once sync.Once
var instance Settings

func GetSettings() *Settings {
	once.Do(func() {
		instance = Settings{}
		flag.StringVar(&instance.dataPath, "data-path", "data", "Path to data directory")
		flag.BoolVar(&instance.debug, "debug", false, "Enable trace debugging")
		flag.Parse()

		instance.accountNumber = "000000000000"
		instance.region = "us-west-2"
	})

	return &instance
}

func (settings *Settings) GetDataPath() string {
	if settings.dataPath[0] == '/' {
		return settings.dataPath
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return filepath.Join(cwd, settings.dataPath)
}

func (settings *Settings) GetArnFragment() string {
	return settings.region + ":" + settings.accountNumber
}

func (settings *Settings) IsDebug() bool {
	return settings.debug
}

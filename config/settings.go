package config

import (
	"flag"
	"sync"
)

type Settings struct {
	dataPath string
}

var once sync.Once
var instance Settings

func GetSettings() *Settings {
	once.Do(func() {
		instance = Settings{}
		flag.StringVar(&instance.dataPath, "data-path", "data", "Path to data directory")
		flag.Parse()
	})

	return &instance
}

func (settings *Settings) GetDataPath() string {
	return settings.dataPath
}

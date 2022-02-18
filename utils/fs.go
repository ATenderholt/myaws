package utils

import (
	"errors"
	"myaws/log"
	"os"
)

func CreateDirs(dirPath string) error {
	log.Debug("Creating directory if necessary %s ...", dirPath)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		msg := log.Error("Unable to create directory %s: %v", dirPath, err)
		return errors.New(msg)
	}

	return nil
}

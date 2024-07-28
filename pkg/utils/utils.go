package utils

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

func SetupLogger(logLevel string) {
	switch strings.ToLower(logLevel) {
	case "debug":
		log.Info("Setting log level to debug")
		log.SetLevel(log.DebugLevel)
	case "info":
		log.Info("Setting log level to info")
		log.SetLevel(log.InfoLevel)
	// No point in logging an Info message for WarnLevel and ErrorLevel as the user probably wants to see warnings and errors only
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
		log.Info("Invalid log level passed, using InfoLevel", "passed", logLevel)
	}
}

// EmptyDir empties a directory by removing all files and directories in it and recreating the directory
func EmptyDir(pathToDir string) error {
	// Remove all files in the directory
	os.RemoveAll(pathToDir)
	// Recreate the directory
	err := os.MkdirAll(pathToDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

package utils

import (
	"os"
	"path/filepath"
	"strings"
	"time"

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
	err := os.MkdirAll(pathToDir, 0644)
	if err != nil {
		return err
	}
	return nil
}

// RollingDir takes a directory path and a number of backups to keep. It is intended to be run before a backup is started.
// It will remove all but the most recent n-1 backups (directories) in the directory. The backup directories are expected to be named in the format `backup-<timestamp>`.
// After making sure the directory has the correct number of backups, it will return the path to what the next backup directory should be named.
func RollingDir(pathToDir string, maxBackups int) (string, error) {
	// Get all directories in the path

	return "", nil
}

// https://stackoverflow.com/questions/8824571/golang-determining-whether-file-points-to-file-or-directory/25567952#25567952

func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

// Function to make a subdirectory in the parent directory with the current time
func CreateTimeBasedDir(parentDir string) (string, error) {
	// https://stackoverflow.com/questions/42217308/go-time-format-how-to-understand-meaning-of-2006-01-02-layout/42217483#42217483
	// 2006: year; 01: month; 02: day; 15: hour; 04: minute; 05: second
	newPath := filepath.Join(parentDir, time.Now().Format("2006-01-02-15-04-05"))
	err := os.MkdirAll(newPath, 0644)

	if err != nil {
		return "", err
	}
	return newPath, nil
}

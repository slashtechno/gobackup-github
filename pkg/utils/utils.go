package utils

import (
	"os"
	"path/filepath"
	"slices"
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

// EmptyDir empties a directory by removing all files and directories in it and recreating the directory.
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
	// https://stackoverflow.com/questions/42217308/go-time-format-how-to-understand-meaning-of-2006-01-02-layout/42217483#42217483
	// 2006: year; 01: month; 02: day; 15: hour; 04: minute; 05: second
	timeFormat := "2006-01-02-15-04-05"

	// Get all directories in the path
	dirs := []string{}

	filesAndDirs, err := os.ReadDir(pathToDir)
	if err != nil {
		return "", err
	}

	for _, fileAndDir := range filesAndDirs {
		if fileAndDir.IsDir() {
			dirs = append(dirs, fileAndDir.Name())
		} else {
			log.Warn("Found a file in the backup directory, ignoring", "file", fileAndDir.Name())
		}
	}

	log.Debug("Found directories", "directories", dirs)

	overLimit := (len(dirs) - maxBackups) + 1

	for i := 0; i < overLimit; i++ {
		err := RemoveOldestDir(&dirs, pathToDir, timeFormat)
		if err != nil {
			return "", err
		}
	}

	// for now, just return the CreateTimeBasedDir even if there are too many directories
	dirPath, err := CreateTimeBasedDir(pathToDir, timeFormat)
	if err != nil {
		return "", err
	}

	return dirPath, nil

}

func RemoveOldestDir(dirNames *[]string, parentDir string, timeFormat string) error {
	// Sort the directories by time
	// https://stackoverflow.com/questions/23121026/how-to-sort-by-time-time/77235904#77235904

	timestampSort := func(i, j string) int {
		timeI, err := time.Parse(timeFormat, i)
		if err != nil {
			return 0
		}
		timeJ, err := time.Parse(timeFormat, j)
		if err != nil {
			return 0
		}

		return timeI.Compare(timeJ)
	}
	slices.SortFunc(*dirNames, timestampSort)

	// Remove the oldest directory
	toRemove := filepath.Join(parentDir, (*dirNames)[0])
	err := os.RemoveAll(toRemove)
	log.Info("Removed oldest backup", "path", toRemove)
	if err != nil {
		return err
	}

	return nil
}

// Function to make a subdirectory in the parent directory with the current time
func CreateTimeBasedDir(parentDir string, timeFormat string) (string, error) {
	newPath := filepath.Join(parentDir, time.Now().Format(timeFormat))
	err := os.MkdirAll(newPath, 0644)

	if err != nil {
		return "", err
	}
	return newPath, nil
}

// If a directory returns true, if it isn't a directory it returns false
// If an error occurs, like the path not existing, it returns the error from os.Stat.
func IsDirectory(path string) (bool, error) {
	// https://stackoverflow.com/questions/8824571/golang-determining-whether-file-points-to-file-or-directory/25567952#25567952
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

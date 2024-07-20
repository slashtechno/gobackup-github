package backup

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/go-github/v63/github"
	"github.com/slashtechno/gobackup-github/cobra/internal"
)

func Backup(config BackupConfig) error {
	// backup
	log.Debug("Backup", "config", config)
	// Get repositories
	repos := []*github.Repository{}
	for _, username := range config.Usernames {
		fetchedRepos, err := GetRepositories(
			FetchConfig{
				Username: username,
				Token:    config.Token,
			},
		)
		if err != nil {
			return err
		}
		repos = append(repos, fetchedRepos.User...)
		repos = append(repos, fetchedRepos.Starred...)
	}

	// Remove duplicates
	noDuplicates := RemoveDuplicateRepositories(repos)
	log.Info("Got repositories", "count", len(noDuplicates))
	for _, repo := range noDuplicates {
		log.Debug("Got repository", "repository", repo.GetFullName())
		err := cloneRepository(repo, config)
		if err != nil {
			return err
		}

	}
	return nil
}

func StartBackup(
	usernames []string,
	token string,
	output string,
	interval string,
) error {
	backupConfig := BackupConfig{
		Usernames: usernames,
		Token:     token,
		Output:    output,
	}

	// https://gobyexample.com/tickers
	if interval != "" {
		log.Info("Starting backup with interval", "interval", interval)

		ticker := time.NewTicker(internal.Viper.GetDuration("interval"))
		defer ticker.Stop()
		var wg sync.WaitGroup
		wg.Add(1)

		// Run backup on start
		err := Backup(backupConfig)
		if err != nil {
			return err
		}
		go func() {
			for range ticker.C {
				err = Backup(backupConfig)
				if err != nil {
					// Don't stop the backup process if one fails
					log.Error("Failed to backup", "error", err)
					continue
				}
			}
		}()
		wg.Wait()
	}
	log.Info("Starting backup")
	err := Backup(backupConfig)
	if err != nil {
		return err
	}
	return nil
}

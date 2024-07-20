package backup

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v63/github"
	"github.com/slashtechno/gobackup-github/cobra/internal"
)

type BackupConfig struct {
	Usernames []string
	InOrg     []string
	Token     string
	Output    string
}

// func GetUsersInOrg(
// 	org string,
// 	client *github.Client,
// ) ([]string, error) {

// }

func Backup(config BackupConfig) error {
	// backup
	log.Debug("Backup", "config", config)

	// Make a client
	// Make an HTTP client that waits if the rate limit is exceeded
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		return err
	}
	// .WithEnterpriseURL could probably be used for something like Gitea
	client := github.NewClient(rateLimiter).WithAuthToken(config.Token)

	// Get users in org
	// usersInOrg := []string{}

	// Get repositories
	repos := []*github.Repository{}
	for _, username := range config.Usernames {

		fetchConfig := &FetchConfig{
			Client:   client,
			Token:    config.Token,
			Username: username,
		}
		fetchedRepos, err := GetRepositories(
			fetchConfig,
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
	inOrg []string,
	token string,
	output string,
	interval string,
) error {
	backupConfig := BackupConfig{
		Usernames: usernames,
		InOrg:     inOrg,
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

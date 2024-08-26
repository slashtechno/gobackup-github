package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"

	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v63/github"
	"github.com/slashtechno/gobackup-github/pkg/utils"
)

type BackupConfig struct {
	Usernames   []string
	InOrg       []string
	BackupStars bool
	Token       string
	Output      string
	// RunType can be `clone`, `fetch`, or `dry-run`
	RunType string
	NtfyUrl string
}

func GetUsersInOrg(
	orgName string,
	client *github.Client,
) ([]string, error) {
	ctx := context.Background()
	// https://pkg.go.dev/github.com/google/go-github/v63@v63.0.0/github#OrganizationsService.Get
	var members []string
	opt := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		membersReturned, resp, err := client.Organizations.ListMembers(ctx, orgName, opt)
		if err != nil {
			return nil, err
		}
		// Get the repository from the starred repository
		for _, m := range membersReturned {
			members = append(members, m.GetLogin())
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return members, nil
}

func Backup(config BackupConfig) error {
	// Make a client
	// Make an HTTP client that waits if the rate limit is exceeded
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		return err
	}
	// .WithEnterpriseURL could probably be used for something like Gitea
	client := github.NewClient(rateLimiter).WithAuthToken(config.Token)

	// Get users in org
	var allUsers []string
	for _, org := range config.InOrg {
		users, err := GetUsersInOrg(org, client)
		if err != nil {
			return err
		}
		allUsers = append(allUsers, users...)
	}
	allUsers = append(allUsers, config.Usernames...)

	// Get repositories
	var repos []*github.Repository
	fetchConfig := &FetchConfig{
		GetStars: config.BackupStars,
		Client:   client,
		Token:    config.Token,
	}
	for _, username := range allUsers {

		fetchConfig.Username = username

		fetchedRepos, err := GetRepositories(
			fetchConfig,
		)
		if err != nil {
			return err
		}
		repos = append(repos, fetchedRepos.User...)
		repos = append(repos, fetchedRepos.Starred...)
	}
	if len(allUsers) == 0 {
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
	log.Info("Deduplicated repositories", "count", len(noDuplicates))
	if config.RunType == "clone" {
		log.Info("Cloning repositories")
		var wg sync.WaitGroup
		errChan := make(chan error, len(noDuplicates))
		bar := progressbar.Default(int64(len(noDuplicates)))

		for _, repo := range noDuplicates {
			wg.Add(1)
			go func(repo *github.Repository) {
				defer wg.Done()

				err := cloneRepository(repo, config)
				if err != nil {
					errChan <- err
					return
				}

				log.Debug("Cloned repository", "repository", repo.GetFullName())
				bar.Add(1)
			}(repo)
		}

		wg.Wait()
		close(errChan)
		for err := range errChan {
			if err != nil {
				return err
			}
		}

	} else if config.RunType == "fetch" {
		var output string
		log.Info("Fetching repositories")
		if filepath.Ext(config.Output) != ".json" {
			// TODO: Make sure directories exist
			output = filepath.Join(config.Output, "repositories.json")
			log.Info("Output file should be a JSON file. Attempting to use `repositories.json` in the output directory", "path", output)
		} else {
			output = config.Output
			log.Debug("Using specified output file", "path", output)
		}

		// Marshal the repositories to JSON
		repoJson, err := json.MarshalIndent(noDuplicates, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(output, repoJson, 0644)
		if err != nil {
			return err
		}
		log.Info("Fetched and saved list of repositories", "path", output)
	} else if config.RunType == "dry-run" {
		repoJson, err := json.MarshalIndent(noDuplicates, "", "  ")
		if err != nil {
			return err
		}
		log.Debug("Dry run - printing repositories to console")
		fmt.Println(string(repoJson))
	} else {
		return fmt.Errorf("invalid run type: %s; must be one of `clone`, `fetch`, or `dry-run`", config.RunType)
	}

	if config.NtfyUrl != "" {
		log.Info("Sending notification", "url", config.NtfyUrl)
		_, err := resty.New().R().SetHeader("Tags", "tada").SetBody("Backup complete").Post(config.NtfyUrl)
		if err != nil {
			return err
		}
	}
	return nil
}

func StartBackup(
	config BackupConfig,
	interval string,
	maxBackups int,
) error {
	backupConfig := config

	// https://gobyexample.com/tickers
	if interval != "" {
		log.Info("Starting backup with interval", "interval", interval)

		parentDir := filepath.Clean(backupConfig.Output)

		if maxBackups < 1 {
			log.Warn("maxBackups must be greater than 0. Setting to 1", "maxBackups", maxBackups)
			maxBackups = 1
		}

		duration, err := time.ParseDuration(interval)
		if err != nil {
			return err
		}

		ticker := time.NewTicker(duration)
		defer ticker.Stop()
		var wg sync.WaitGroup

		// Run backup on start
		backupConfig.Output, err = utils.RollingDir(parentDir, maxBackups)
		if err != nil {
			return err
		}
		err = Backup(backupConfig)
		if err != nil {
			return err
		}
		errChan := make(chan error)
		go func() {
			defer close(errChan)
			// The backup will not be concurrent if the backup process takes longer than the interval
			for range ticker.C {
				wg.Add(1)

				backupConfig.Output, err = utils.RollingDir(parentDir, maxBackups)
				if err != nil {
					errChan <- err
					return
				}
				err = Backup(backupConfig)
				if err != nil {
					errChan <- err
					return
				}

				wg.Done()
			}

		}()
		wg.Wait()
		// Handle errors from the goroutine
		for err := range errChan {
			if err != nil {
				return err
			}
		}
	}
	log.Info("Starting backup")
	err := Backup(backupConfig)
	if err != nil {
		return err
	}
	return nil
}

package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
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
		for _, repo := range noDuplicates {
			err := cloneRepository(repo, config)
			if err != nil {
				return err
			}
			log.Info("Cloned repository", "repository", repo.GetFullName())

		}
	} else if config.RunType == "fetch" {
		var output string
		log.Info("Fetching repositories")
		if filepath.Ext(config.Output) != ".json" {
			output = filepath.Join(config.Output, "repositories.json")
			log.Warn("Output file should be a JSON file. Attempting to use `repositories.json` in the output directory", "path", output)
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
		log.Info("Dry run - printing repositories to console")
		fmt.Println(string(repoJson))
	} else {
		return fmt.Errorf("invalid run type: %s; must be one of `clone`, `fetch`, or `dry-run`", config.RunType)
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

		log.Info("Emptying output directory", "output", backupConfig.Output)
		cleanedPath := filepath.Clean(backupConfig.Output)
		err := utils.EmptyDir(cleanedPath)
		if err != nil {
			return err
		}

		duration, err := time.ParseDuration(interval)
		if err != nil {
			return err
		}

		ticker := time.NewTicker(duration)
		defer ticker.Stop()
		var wg sync.WaitGroup

		// Run backup on start
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
				log.Info("Emptying output directory", "output", backupConfig.Output)
				cleanedPath := filepath.Clean(backupConfig.Output)
				err := utils.EmptyDir(cleanedPath)
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

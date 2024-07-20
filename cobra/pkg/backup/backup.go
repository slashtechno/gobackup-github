package backup

import (
	"context"
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
	repos := []*github.Repository{}
	for _, username := range allUsers {

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
	if len(allUsers) == 0 {
		fetchConfig := &FetchConfig{
			Client: client,
			Token:  config.Token,
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
	log.Info("Deduplicated repositories", "count", len(noDuplicates))
	for _, repo := range noDuplicates {
		err := cloneRepository(repo, config)
		if err != nil {
			return err
		}
		log.Info("Cloned repository", "repository", repo.GetFullName())

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

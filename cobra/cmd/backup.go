/*
Copyright © 2024 Angad Behl
*/
package cmd

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v63/github"

	"github.com/slashtechno/gobackup-github/cobra/internal"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup a GitHub user",
	Long: `Backup either the authenticated user or a specified GitHub user.
	Backing up the authenticated user clones private repositories as well.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		startBackup(
			internal.Viper.GetString("username"),
			internal.Viper.GetString("token"),
			internal.Viper.GetString("output"),
			internal.Viper.GetString("interval"),
		)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.PersistentFlags().StringP("username", "u", "", "GitHub username to backup. Leave blank to backup the authenticated user")
	internal.Viper.BindPFlag("username", backupCmd.PersistentFlags().Lookup("username"))
	internal.Viper.SetDefault("username", "")

	backupCmd.PersistentFlags().StringP("token", "t", "", "GitHub token")
	internal.Viper.BindPFlag("token", backupCmd.PersistentFlags().Lookup("token"))
	internal.Viper.SetDefault("token", "")

	backupCmd.PersistentFlags().StringP("output", "o", "", "Output directory")
	internal.Viper.BindPFlag("output", backupCmd.PersistentFlags().Lookup("output"))
	internal.Viper.SetDefault("output", "backup")

	backupCmd.Flags().StringP("interval", "i", "", "Interval to check for new content")
	internal.Viper.BindPFlag("interval", backupCmd.Flags().Lookup("interval"))
}

func startBackup(
	username string,
	token string,
	output string,
	interval string,
) error {
	backupConfig := BackupConfig{
		Username: username,
		Token:    token,
		Output:   output,
	}

	// https://gobyexample.com/tickers
	if interval != "" {
		log.Info("Starting backup with interval", "interval", interval)

		ticker := time.NewTicker(internal.Viper.GetDuration("interval"))
		defer ticker.Stop()
		var wg sync.WaitGroup
		wg.Add(1)

		// Run backup on start
		err := backup(backupConfig)
		if err != nil {
			return err
		}
		go func() {
			for range ticker.C {
				err = backup(backupConfig)
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
	err := backup(backupConfig)
	if err != nil {
		return err
	}
	return nil
}

type BackupConfig struct {
	Username string
	Token    string
	Output   string
}

func backup(config BackupConfig) error {
	// backup
	log.Debug("Backup", "config", config)

	// Make an HTTP client that waits if the rate limit is exceeded
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		return err
	}

	// .WithEnterpriseURL could probably be used for something like Gitea
	client := github.NewClient(rateLimiter).WithAuthToken(config.Token)
	ctx := context.Background()

	// Get the user
	// https://pkg.go.dev/github.com/google/go-github/v63/github#User
	// If the username is an empty string, Users.Get will return the authenticated user
	user, _, err := client.Users.Get(ctx, config.Username)
	if err != nil {
		return err
	}
	username := user.GetLogin()
	log.Info("Backing up user", "username", username)

	// https://github.com/google/go-github?tab=readme-ov-file#pagination
	listOptions := github.ListOptions{PerPage: 100}

	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-repositories-for-a-user
	// https://pkg.go.dev/github.com/google/go-github/v63@v63.0.0/github#RepositoriesService.ListByUser
	userRepos := []*github.Repository{}
	if config.Username == "" {
		opt := &github.RepositoryListByAuthenticatedUserOptions{
			ListOptions: listOptions,
		}
		for {
			repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, opt)
			if err != nil {
				return err
			}
			userRepos = append(userRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	} else {
		opt := &github.RepositoryListByUserOptions{
			ListOptions: listOptions,
		}
		for {
			repos, resp, err := client.Repositories.ListByUser(ctx, username, opt)
			if err != nil {
				return err
			}
			userRepos = append(userRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}

	log.Info("Fetched user's repositories", "count", len(userRepos))

	// Get the starred repositories
	// client.Activity.ListStarred(ctx, username, nil)
	// Deal with pagination
	opt := &github.ActivityListStarredOptions{
		ListOptions: listOptions,
	}
	var starredRepos []*github.Repository
	for {
		repos, resp, err := client.Activity.ListStarred(ctx, username, opt)
		if err != nil {
			return err
		}
		// Get the repository from the starred repository
		for _, repo := range repos {
			starredRepos = append(starredRepos, repo.GetRepository())
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	log.Info("Fetched user's starred repositories", "count", len(starredRepos))

	allRepos := append(userRepos, starredRepos...)
	for _, repo := range allRepos {
		log.Debug("Got repository", "repository", repo.GetFullName())
	}

	return nil
}

package backup

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v63/github"
)

type FetchConfig struct {
	GetStars bool
	Client   *github.Client
	Username string
	Token    string
}

type Repositories struct {
	User    []*github.Repository
	Starred []*github.Repository
}

func cloneRepository(repo *github.Repository, config BackupConfig) error {
	// Set the output directory
	outputDirectory := filepath.Join(config.Output, repo.GetFullName())
	// Clone the repository
	_, err := git.PlainClone(outputDirectory, false, &git.CloneOptions{
		URL: repo.GetCloneURL(),
		Auth: &http.BasicAuth{
			// Username: config.Username,
			Username: config.Token,
			Password: config.Token,
		},
		SingleBranch:      false, // False by default
		RecurseSubmodules: git.SubmoduleRescursivity(config.RecurseSubmodules),
	})
	if err != nil {
		return err
	}
	return nil
}

// Get both starred and user repositories, remove duplicates, and return them as a Repositories struct.
// Takes a BackupConfig struct as an argument. BackupConfig requires a username and token but not output.
func GetRepositories(config *FetchConfig) (*Repositories, error) {
	reposToReturn := &Repositories{}

	ctx := context.Background()
	if config.Client == nil {
		return nil, fmt.Errorf("client is nil")
	}
	client := config.Client

	// Get the user
	// https://pkg.go.dev/github.com/google/go-github/v63/github#User
	// If the username is an empty string, Users.Get will return the authenticated user
	user, _, err := client.Users.Get(ctx, config.Username)
	if err != nil {
		return nil, err
	}
	username := user.GetLogin()
	log.Info("Fetching repositories for user", "username", username)

	// https://github.com/google/go-github?tab=readme-ov-file#pagination
	listOptions := github.ListOptions{PerPage: 100}

	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-repositories-for-a-user
	// https://pkg.go.dev/github.com/google/go-github/v63@v63.0.0/github#RepositoriesService.ListByUser
	var userRepos []*github.Repository
	if config.Username == "" {
		opt := &github.RepositoryListByAuthenticatedUserOptions{
			ListOptions: listOptions,
		}
		for {
			repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, opt)
			if err != nil {
				return nil, err
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
				return nil, err
			}
			userRepos = append(userRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}
	log.Debug("Fetched user's repositories", "count", len(userRepos), "username", username)
	reposToReturn.User = userRepos

	// Get the starred repositories
	if config.GetStars {
		// client.Activity.ListStarred(ctx, username, nil)
		// Deal with pagination
		opt := &github.ActivityListStarredOptions{
			ListOptions: listOptions,
		}
		var starredRepos []*github.Repository
		for {
			repos, resp, err := client.Activity.ListStarred(ctx, username, opt)
			if err != nil {
				return nil, err
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
		log.Debug("Fetched user's starred repositories", "count", len(starredRepos), "username", username)
		reposToReturn.Starred = starredRepos

		// allRepos := append(userRepos, starredRepos...)
		// for _, repo := range allRepos {
		// 	log.Debug("Got repository", "repository", repo.GetFullName())
		// }
	}

	return reposToReturn, nil

}

// Go through a list of repositories and remove duplicates.
func RemoveDuplicateRepositories(repositories []*github.Repository,
) []*github.Repository {
	var noDuplicates []*github.Repository

	for _, repo := range repositories {

		// FOR DEBUGGING
		if repo.GetFullName() == "yourselfhosted/slash" {
			log.Debug("Found slash")
		}

		found := false
		for _, added := range noDuplicates {
			if repo.GetFullName() == added.GetFullName() {
				found = true
				log.Debug("Found duplicate", "repository", repo.GetFullName())
				break
			}
		}
		if !found {
			noDuplicates = append(noDuplicates, repo)
		}
	}
	return noDuplicates
}

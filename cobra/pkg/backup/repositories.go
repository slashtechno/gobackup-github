package backup

import (
	"context"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v63/github"
)

type FetchConfig struct {
	Username string
	Token    string
}

type BackupConfig struct {
	Usernames []string
	Token     string
	Output    string
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
	})
	if err != nil {
		return err
	}
	log.Info("Cloned repository", "repository", repo.GetFullName())
	return nil
}

// Get both starred and user repositories, remove duplicates, and return them as a Repositories struct.
// Takes a BackupConfig struct as an argument. BackupConfig requires a username and token but not output.
func GetRepositories(config FetchConfig) (*Repositories, error) {
	// Make an HTTP client that waits if the rate limit is exceeded
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		return nil, err
	}

	// .WithEnterpriseURL could probably be used for something like Gitea
	client := github.NewClient(rateLimiter).WithAuthToken(config.Token)
	ctx := context.Background()

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
	userRepos := []*github.Repository{}
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
	log.Info("Fetched user's starred repositories", "count", len(starredRepos))

	// allRepos := append(userRepos, starredRepos...)
	// for _, repo := range allRepos {
	// 	log.Debug("Got repository", "repository", repo.GetFullName())
	// }

	return &Repositories{
		User:    userRepos,
		Starred: starredRepos,
	}, nil

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

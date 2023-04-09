package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type repoStruct struct {
	HTMLURL     string `json:"html_url"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}
type gistStruct struct {
	ID         string `json:"id"`
	GitPullURL string `json:"git_pull_url"`

	Public      bool   `json:"public"`
	Description string `json:"description"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

var backedUp = map[string]map[string]map[string]string{}

var dateToday = time.Now().Format("01-02-2006")

var (
	currentDirectory, _ = os.Getwd()
	backupDirectory     = filepath.Join(currentDirectory, "github-backup-from-"+dateToday)
)

type BackupCmd struct {
	Targets    []string `arg:"positional" help:"What to backup. Options are: repos; stars; gists"`
	CreateList bool     `arg:"-c,--create-list" help:"Create a list of repositories"`
	NoClone    bool     `arg:"-n,--no-clone" help:"Don't clone the repositories"`
	// Add backup dir
}

var args struct {
	Backup *BackupCmd `arg:"subcommand:backup" help:"Backup your GitHub data"`

	// Environment variables
	Token string `arg:"env:GITHUB_TOKEN" help:"GitHub token"`

	// Misc flags
	LogLevel string `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\", \"warning\", \"error\", or \"fatal\"" default:"info"`
	LogColor bool   `arg:"--log-color, env:LOG_COLOR" help:"Force colored logs" default:"false"`
}

func main() {
	fmt.Println("Today is " + dateToday)
	// Command line stuff
	godotenv.Load()
	arg.MustParse(&args)

	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{PadLevelText: true, DisableQuote: true, ForceColors: args.LogColor, DisableColors: !args.LogColor})
	if args.LogLevel == "debug" {
		logrus.SetLevel(logrus.DebugLevel)
		// Enable line numbers in debug logs - Doesn't help too much since a fatal error still needs to be debugged
		logrus.SetReportCaller(true)
	} else if args.LogLevel == "info" {
		logrus.SetLevel(logrus.InfoLevel)
	} else if args.LogLevel == "warning" {
		logrus.SetLevel(logrus.WarnLevel)
	} else if args.LogLevel == "error" {
		logrus.SetLevel(logrus.ErrorLevel)
	} else if args.LogLevel == "fatal" {
		logrus.SetLevel(logrus.FatalLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	var err error
	switch {
	case args.Backup != nil:
		if len(args.Backup.Targets) == 0 {
			logrus.Fatal("No targets specified")
		}

		backedUp, err = backupRepos(args.Backup.Targets, !args.Backup.NoClone, args.Token)
		if err != nil {
			logrus.Fatal(err)
		}
	default:
		mainMenu()
	}
	// List creation
	if args.Backup.CreateList {
		if err := createList(backedUp); err != nil {
			logrus.Fatal(err)
		}
	}
}

func createList(repos map[string]map[string]map[string]string) error {
	logrus.Info("Creating list of repositories")
	// Write the repos to a JSON file
	if _, err := os.Stat(backupDirectory); !os.IsExist(err) {
		// Maybe just make a file instead of a directory, for now, a directory is fine
		if err := os.MkdirAll(backupDirectory, 0600); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(filepath.Join(backupDirectory, "list.json"), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(repos); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}

func backupRepos(repoTypes []string, clone bool, token string) (map[string]map[string]map[string]string, error) {
	url := "https://api.github.com/user"
	response, err := ghRequest(url, token)
	if err != nil {
		return nil, err
	}
	content, err := responseContent(response.Body)
	if err != nil {
		return nil, err
	}
	user := gjson.Get(content, "login")

	// Repos is a map of repo names. This is just to store repos in {repoTypes: [<id or full name>: {"description": repoDescription, "url": <html_url or git_pull_url>}, <id or full name>...]} format. So a map of maps
	repos := map[string]map[string]map[string]string{}

	for _, repoType := range repoTypes {
		// Declaring variables so that they can be used in the switch statement and outside of it
		var (
			// repoSlice is a slice of repoStructs
			repoSlice   []repoStruct
			gistSlice   []gistStruct
			url         string
			neededPages int
		)

		// Create a map for the repo type ( to prevent a panic: assignment to entry in nil map)
		repos[repoType] = map[string]map[string]string{}

		if repoType == "repos" {
			url = "https://api.github.com/user/repos?per_page=100&page="
			neededPages, err = calculateNeededPages("repos", token)
			if err != nil {
				return nil, err
			}
			logrus.Info("Getting list of repositories for " + user.String())
		} else if repoType == "stars" {
			url = "https://api.github.com/user/starred?per_page=100&page="
			neededPages, err = calculateNeededPages("stars", token)
			if err != nil {
				return nil, err
			}
			logrus.Info("Getting list of starred repositories for " + user.String())
		} else if repoType == "gists" {
			url = "https://api.github.com/gists?per_page=100&page="
			neededPages, err = calculateNeededPages("gists", token)
			if err != nil {
				return nil, err
			}
		}
		for i := 1; i <= neededPages; i++ {
			if repoType == "gists" {
				response, err := ghRequest(url+strconv.Itoa(i), token)
				if err != nil {
					return nil, err
				}
				gistJSON, err := responseContent(response.Body)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal([]byte(gistJSON), &gistSlice); err != nil {
					return nil, err
				}
				for i := 0; i < len(gistSlice); i++ {
					owner := gistSlice[i].Owner.Login
					if clone {
						cloneDirectory := filepath.Join(backupDirectory, repoType, owner, gistSlice[i].ID)
						logrus.Infof("Cloning %v (iteration %v) to %v\n", gistSlice[i].ID, i+1, cloneDirectory)
						logrus.Debug("URL: " + gistSlice[i].GitPullURL)
						_, err = git.PlainClone(cloneDirectory, false, &git.CloneOptions{
							URL: gistSlice[i].GitPullURL,
							Auth: &githttp.BasicAuth{
								Username: user.String(),
								Password: token,
							},
						})
						if err != nil {
							return nil, err
						}
					}
					// Set the description and url of the gist
					repos[repoType][gistSlice[i].ID] = map[string]string{}
					// There is no title for gists, so just use the ID and don't set the name
					repos[repoType][gistSlice[i].ID]["description"] = gistSlice[i].Description
					repos[repoType][gistSlice[i].ID]["url"] = gistSlice[i].GitPullURL
				}
			} else if repoType == "repos" || repoType == "stars" {
				response, err := ghRequest(url+strconv.Itoa(i), token)
				if err != nil {
					return nil, err
				}
				repoJSON, err := responseContent(response.Body)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal([]byte(repoJSON), &repoSlice); err != nil {
					return nil, err
				}
				for i := 0; i < len(repoSlice); i++ {
					owner := repoSlice[i].Owner.Login
					if clone {
						cloneDirectory := filepath.Join(backupDirectory, repoType, owner, repoSlice[i].Name)
						logrus.Infof("Cloning %v (iteration %v) to %v\n", repoSlice[i].Name, i+1, cloneDirectory)
						logrus.Debug("URL: " + repoSlice[i].HTMLURL)
						_, err = git.PlainClone(cloneDirectory, false, &git.CloneOptions{
							URL: repoSlice[i].HTMLURL,
							Auth: &githttp.BasicAuth{
								Username: user.String(), // anything except an empty string
								Password: token,
							},
						})
						if err != nil {
							return nil, err
						}

					}
					// Set the description and url of the repo
					repos[repoType][repoSlice[i].Name] = map[string]string{}
					// Perhaps use <user uuid>/<repo uuid> instead of <user>/<repo>?
					repos[repoType][repoSlice[i].Name]["name"] = repoSlice[i].Name
					repos[repoType][repoSlice[i].Name]["description"] = repoSlice[i].Description
					repos[repoType][repoSlice[i].Name]["url"] = repoSlice[i].HTMLURL
				}
			}
		}
	}
	return repos, nil
}

func calculateNeededPages(whichRepos string, token string) (int, error) {
	perPage := 10
	if whichRepos == "repos" {
		response, err := ghRequest("https://api.github.com/user", token)
		if err != nil {
			return 0, err
		}
		json, err := responseContent(response.Body)
		if err != nil {
			return 0, err
		}
		publicRepos := gjson.Get(json, "public_repos")
		privateRepos := gjson.Get(json, "total_private_repos")
		totalRepos := publicRepos.Num + privateRepos.Num
		logrus.Info("Total repositories: " + strconv.Itoa(int(totalRepos)))
		neededPages := math.Ceil(totalRepos / float64(perPage))
		logrus.Info("Total pages needed:" + strconv.Itoa(int(neededPages)))
		return int(neededPages), nil
	} else if whichRepos == "stars" {
		total := 0
		var starSlice []repoStruct
		var pageNumber int
		// If the length of the slice is 0, the request prior to the one just made was the last page
		for len(starSlice) == perPage || pageNumber == 0 {
			pageNumber++
			url := "https://api.github.com/user/starred?page=" + strconv.Itoa(pageNumber) + "&per_page=" + strconv.Itoa(perPage)
			requestContent, err := ghRequest(url, loadToken())
			if err != nil {
				return 0, err
			}
			starJSON, err := responseContent(requestContent.Body)
			if err != nil {
				return 0, err
			}
			if err := json.Unmarshal([]byte(starJSON), &starSlice); err != nil {
				return 0, err
			}
			total += len(starSlice)
		}
		logrus.Info("Total starred repositories: " + strconv.Itoa(total))
		logrus.Info("Total pages needed:" + strconv.Itoa(pageNumber))
		return pageNumber, nil
	} else if whichRepos == "gists" {
		total := 0
		var gistSlice []gistStruct
		var pageNumber int
		for len(gistSlice) == perPage || pageNumber == 0 {
			pageNumber++
			url := "https://api.github.com/gists?page=" + strconv.Itoa(pageNumber) + "&per_page=" + strconv.Itoa(perPage)
			requestContent, err := ghRequest(url, loadToken())
			if err != nil {
				return 0, err
			}
			gistJSON, err := responseContent(requestContent.Body)
			if err != nil {
				return 0, err
			}
			if err := json.Unmarshal([]byte(gistJSON), &gistSlice); err != nil {
				return 0, err
			}
			total += len(gistSlice)
		}
		logrus.Info("Total gists: " + strconv.Itoa(total))
		logrus.Info("Total pages needed: " + strconv.Itoa(pageNumber))
		return pageNumber, nil
	}
	// This functions as an else statement since the function will return before this point if the argument is valid.
	return 0, errors.New("invalid argument (must be \"repos\", \"stars\", or \"gists\")")
}

func responseContent(responseBody io.ReadCloser) (string, error) {
	bytes, err := io.ReadAll(responseBody)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ghRequest(url, token string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	// req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Accept", "testing-github-api")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		if response.StatusCode == 401 {
			return nil, errors.New("unauthorized, check your token")
		} else {
			return nil, errors.New("something went wrong, status code is not \"200 OK\", response status code: " + response.Status)
		}

	}
	return response, nil
}

func loadToken() string {
	return args.Token
}

// Menus

func mainMenu() {
	fmt.Println(`1) Backup repositories
2) Exit`)
	reader := bufio.NewReader(os.Stdin)
	menuSelection, _ := reader.ReadString('\n')
	menuSelection = strings.TrimSpace(menuSelection)
	switch menuSelection {
	case "1":
		backupMenu()
	case "2":
		os.Exit(0)
	}
}

func backupMenu() {
	fmt.Println(`What should this program backup?
1) Your public and private repositories
2) Your starred repositories
3) Both`)
	reader := bufio.NewReader(os.Stdin)
	backupSelection, _ := reader.ReadString('\n')
	backupSelection = strings.TrimSpace(backupSelection)
	switch backupSelection {
	case "1":
		backupRepos([]string{"repos"}, true, args.Token)
	case "2":
		backupRepos([]string{"stars"}, true, args.Token)
	case "3":
		backupRepos([]string{"repos", "stars"}, true, args.Token)
	default:
		logrus.Fatalln("Invalid selection")
	}
}

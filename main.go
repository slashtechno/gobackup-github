package main

import (
	"bufio"
	"encoding/json"
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
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
	Owner   struct {
		Login string `json:"login"`
	} `json:"owner"`
}

var backedUp = map[string]map[string]string{}

var dateToday = time.Now().Format("01-02-2006")

var (
	currentDirectory, _ = os.Getwd()
	backupDirectory     = filepath.Join(currentDirectory, "github-backup-from-"+dateToday)
)

type BackupCmd struct {
	Targets    []string `arg:"positional" help:"What to backup. Options are: repos, stars"`
	CreateList bool     `arg:"-c,--create-list" help:"Create a list of repositories"`
	NoClone    bool     `arg:"-n,--no-clone" help:"Don't clone the repositories"`
	// Add backup dir
}

var args struct {
	Backup *BackupCmd `arg:"subcommand:backup" help:"Backup your GitHub data"`
	// Misc flags
	LogLevel string `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\", \"warning\", \"error\", or \"fatal\"" default:"info"`
	LogColor bool   `arg:"--log-color, env:LOG_COLOR" help:"Force colored logs" default:"false"`
}

func main() {
	logrus.Info("Today is " + dateToday)
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

	switch {
	case args.Backup != nil:
		if len(args.Backup.Targets) == 0 {
			logrus.Fatal("No targets specified")
		}
		backedUp = backupRepos(args.Backup.Targets, !args.Backup.NoClone)
	default:
		mainMenu()
	}
	// List creation
	if args.Backup.CreateList {
		createList(backedUp)
	}
}

func createList(repos map[string]map[string]string) {
	logrus.Info("Creating list of repositories")
	// Write the repos to a JSON file
	if _, err := os.Stat(backupDirectory); !os.IsExist(err) {
		// Maybe just make a file instead of a directory, for now, a directory is fine
		err = os.MkdirAll(backupDirectory, 0644)
		checkNilErr(err)
	}
	file, err := os.OpenFile(filepath.Join(backupDirectory, "list.json"), os.O_CREATE|os.O_WRONLY, 0644)
	checkNilErr(err)
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(repos)
	checkNilErr(err)
}

func backupRepos(repoTypes []string, clone bool) map[string]map[string]string {
	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")

	// Repos is a map of repo names. This is just to store repos in {repoTypes: [repoName: repoURl, repoName: repoURL]} format. So a map of maps
	// I Noticed map[string]any can't be used because it can't be accessed using a[b][c] = d. I also might just be doing something wrong
	repos := map[string]map[string]string{}

	for _, repoType := range repoTypes {
		// Declaring variables so that they can be used in the switch statement and outside of it
		var (
			// repoSlice is a slice of repoStructs
			repoSlice   []repoStruct
			url         string
			neededPages int
		)

		// Create a map for the repo type ( to prevent a panic: assignment to entry in nil map)
		repos[repoType] = map[string]string{}

		switch repoType {
		case "repos":
			url = "https://api.github.com/user/repos?per_page=100&page="
			neededPages = calculateNeededPages("repos")
			logrus.Info("Getting list of repositories for " + user.String())
		case "stars":
			url = "https://api.github.com/user/starred?per_page=100&page="
			neededPages = calculateNeededPages("stars")
			logrus.Info("Getting list of starred repositories for " + user.String())
		default:
			logrus.Fatal("Invalid repo type: " + repoType)
		}
		for i := 1; i <= neededPages; i++ {

			repoJSON := responseContent(ghRequest(url + strconv.Itoa(i)).Body)
			err := json.Unmarshal([]byte(repoJSON), &repoSlice)
			checkNilErr(err)
			for i := 0; i < len(repoSlice); i++ {
				owner := repoSlice[i].Owner.Login
				if clone {
					cloneDirectory := filepath.Join(backupDirectory, repoType, owner, repoSlice[i].Name)
					logrus.Infof("Cloning %v (iteration %v) to %v\n", repoSlice[i].Name, i+1, cloneDirectory)
					_, err = git.PlainClone(cloneDirectory, false, &git.CloneOptions{
						URL: repoSlice[i].HTMLURL,
						Auth: &githttp.BasicAuth{
							Username: user.String(), // anything except an empty string
							Password: loadToken(),
						},
					})
					checkNilErr(err)
				}
				repos[repoType][repoSlice[i].Name] = repoSlice[i].HTMLURL
			}
		}
	}
	return repos
}

func calculateNeededPages(whichRepos string) int {
	if whichRepos == "repos" {
		response := ghRequest("https://api.github.com/user")
		json := responseContent(response.Body)
		publicRepos := gjson.Get(json, "public_repos")
		privateRepos := gjson.Get(json, "total_private_repos")
		totalRepos := publicRepos.Num + privateRepos.Num
		logrus.Info("Total repositories: " + strconv.Itoa(int(totalRepos)))
		neededPages := math.Ceil(totalRepos / 100)
		logrus.Info("Total pages needed:" + strconv.Itoa(int(neededPages)))
		return int(neededPages)
	} else if whichRepos == "stars" {
		var starSlice []repoStruct
		pageNumber := 1
		perPage := 100
		starJSON := responseContent(ghRequest("https://api.github.com/user/starred?page=1&per_page=" + strconv.Itoa(perPage)).Body)
		err := json.Unmarshal([]byte(starJSON), &starSlice)
		checkNilErr(err)
		for len(starSlice) != 0 { // if len(starSlice) == perPage {
			pageNumber++
			starJSON := responseContent(ghRequest("https://api.github.com/user/starred?page=" + strconv.Itoa(pageNumber) + "&per_page=" + strconv.Itoa(perPage)).Body)
			err := json.Unmarshal([]byte(starJSON), &starSlice)
			checkNilErr(err)

		}
		return pageNumber
	} else {
		return 0
	}
}

func responseContent(responseBody io.ReadCloser) string {
	bytes, err := io.ReadAll(responseBody)
	checkNilErr(err)
	return string(bytes)
}

func ghRequest(url string) *http.Response {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	checkNilErr(err)
	req.Header.Set("Authorization", "token "+loadToken())
	// req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Accept", "testing-github-api")
	response, err := http.DefaultClient.Do(req)
	checkNilErr(err)
	if response.StatusCode != 200 {
		logrus.Fatal("Something went wrong, status code is not \"200 OK\"")
		logrus.Fatal("Response status code: " + response.Status)
		if response.StatusCode == 401 {
			logrus.Fatal("Unauthorized, check your token")
		}

	}
	return response
}

func loadToken() string {
	return os.Getenv("GITHUB_TOKEN")
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
		backupRepos([]string{"repos"}, true)
	case "2":
		backupRepos([]string{"stars"}, true)
	case "3":
		backupRepos([]string{"repos", "stars"}, true)
	default:
		logrus.Fatalln("Invalid selection")
	}
}

func checkNilErr(err any) {
	if err != nil {
		logrus.Fatal(err)
	}
}

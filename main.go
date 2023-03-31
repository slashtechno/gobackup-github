package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

var (
	currentDirectory, _ = os.Getwd()
	backupDirectory     = filepath.Join(currentDirectory, "github-backup-"+dateToday)
)

var backedUp = map[string]any{}

var dateToday = time.Now().Format("01-02-2006")

type BackupCmd struct {
	Targets    []string `arg:"positional" help:"What to backup. Options are: repos, stars, gists"`
	CreateList bool     `arg:"-c,--create-list" help:"Create a list of repositories"`
	NoClone    bool     `arg:"-n,--no-clone" help:"Don't clone the repositories"`
}

var args struct {
	Backup *BackupCmd `arg:"subcommand:backup" help:"Backup your GitHub data"`
	// Misc flags
	LogLevel string `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\", \"warning\", \"error\", or \"fatal\"" default:"info"`
	LogColor bool   `arg:"--log-color, env:LOG_COLOR" help:"Force colored logs" default:"false"`
}

func main() {
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
		for _, target := range args.Backup.Targets {
			switch target {
			case "repos":
				backedUp["repos"] = backupRepos(!args.Backup.NoClone)
			case "stars":
				backedUp["stars"] = backupStars(!args.Backup.NoClone)
			case "gists":
				// backedUp["gists"] =  backupGists()
			default:
				logrus.Fatal("Invalid target: " + target)
			}
		}
	default:
		mainMenu()
	}
	// List creation
	if args.Backup.CreateList {
		createList(backedUp)
	}
}

func createList(repos map[string]any) {
	logrus.Info("Creating list of repositories")
	// Write the repos to a JSON file
	file, err := os.OpenFile(filepath.Join(backupDirectory, "list.json"), os.O_CREATE|os.O_WRONLY, 0644)
	checkNilErr(err)
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(repos)
	checkNilErr(err)
}

func backupRepos(clone bool) map[string]any {
	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	logrus.Info("Getting list of repositories")
	var repoSlice []repoStruct
	repos := map[string]any{}
	neededPages := calculateNeededPages("userRepos")
	logrus.Info("Getting list of repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/repos?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &repoSlice)
		checkNilErr(err)
		for i := 0; i < len(repoSlice); i++ {
			owner := repoSlice[i].Owner.Login
			if clone {
				cloneDirectory := filepath.Join(backupDirectory, "your-repos", owner+"_"+repoSlice[i].Name)
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
			repos[repoSlice[i].Name] = repoSlice[i].HTMLURL
		}
	}
	return repos
}

func backupStars(clone bool) map[string]any {
	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	logrus.Info("User: " + user.String())
	var starSlice []repoStruct
	stars := map[string]any{}
	neededPages := calculateNeededPages("userStars")
	logrus.Info("Getting list of starred repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/starred?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &starSlice)
		checkNilErr(err)
		for i := 0; i < len(starSlice); i++ {
			owner := starSlice[i].Owner.Login
			if clone {
				cloneDirectory := filepath.Join(backupDirectory, "your-stars", owner+"_"+starSlice[i].Name)
				logrus.Infof("Cloning %v (iteration %v) to %v\n", starSlice[i].Name, i, cloneDirectory)
				_, err = git.PlainClone(cloneDirectory, false, &git.CloneOptions{
					URL: starSlice[i].HTMLURL,
					Auth: &githttp.BasicAuth{
						Username: user.String(), // anything except an empty string
						Password: loadToken(),
					},
				})
				checkNilErr(err)
			}
			stars[starSlice[i].Name] = starSlice[i].HTMLURL
		}
	}
	return stars
}

func calculateNeededPages(whichRepos string) int {
	if whichRepos == "userRepos" {
		response := ghRequest("https://api.github.com/user")
		json := responseContent(response.Body)
		publicRepos := gjson.Get(json, "public_repos")
		privateRepos := gjson.Get(json, "total_private_repos")
		totalRepos := publicRepos.Num + privateRepos.Num
		logrus.Info("Total repositories: " + strconv.Itoa(int(totalRepos)))
		neededPages := math.Ceil(totalRepos / 100)
		logrus.Info("Total pages needed:" + strconv.Itoa(int(neededPages)))
		return int(neededPages)
	} else if whichRepos == "userStars" {
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
	req, err := http.NewRequest(http.MethodGet, url, nil)
	checkNilErr(err)
	req.Header.Set("Authorization", "token "+loadToken())
	// req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Accept", "testing-github-api")
	response, err := http.DefaultClient.Do(req)
	checkNilErr(err)
	if response.StatusCode != 200 {
		log.Println("Something went wrong, status code is not \"200 OK\"")
		log.Println("Response status code: " + response.Status)
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
		backedUp["repos"] = backupRepos(true)
	case "2":
		backedUp["stars"] = backupStars(true)
	case "3":
		backedUp["repos"] = backupRepos(true)
		backedUp["stars"] = backupStars(true)
	}
}

func checkNilErr(err any) {
	if err != nil {
		// log.Fatalln("Error:\n%v\n", err)
		logrus.Fatal(err)
	}
}

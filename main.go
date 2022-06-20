package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/joho/godotenv"
	"github.com/tidwall/gjson"
)

type repoStruct struct {
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
	Owner   struct {
		Login string `json:"login"`
	} `json:"owner"`
}

var currentDirectory, _ = os.Getwd()
var dateToday = time.Now().Format("01-02-2006")
var allRepos = map[string]any{}

func main() {
	godotenv.Load()
	repoFlag := flag.Bool("backup-repos", false, "Set this flag to backup your repositories and skip the interactive UI (can be combined with backup-stars)")
	starFlag := flag.Bool("backup-stars", false, "Set this flag to backup your starred repositoriesand skip the interactive UI (can be combined with backup-repos)")
	flag.Parse()
	if *repoFlag && *starFlag {
		backupRepos(true)
		backupStars(true)
	} else if *repoFlag {
		backupRepos(true)
	} else if *starFlag {
		backupStars(true)
	} else {
		mainMenu()
	}
}

func backupRepos(clone bool) map[string]any {

	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	fmt.Println("User: " + user.String())
	var repoSlice []repoStruct
	repos := map[string]any{}
	neededPages := calculateNeededPages("userRepos")
	fmt.Println("Getting list of repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/repos?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &repoSlice)
		if err != nil {
			fmt.Printf("Error:\n%v\n", err)
		}
		for i := 0; i < len(repoSlice); i++ {
			owner := repoSlice[i].Owner.Login
			if clone {
				cloneDirectory := filepath.Dir(currentDirectory + "/github-backup-" + dateToday + "/your-repos/" + owner + "_" + repoSlice[i].Name + "/")
				fmt.Printf("Cloning %v (iteration %v) to %v\n", repoSlice[i].Name, i+1, cloneDirectory)
				_, err = git.PlainClone(cloneDirectory, false, &git.CloneOptions{
					URL: repoSlice[i].HTMLURL,
					Auth: &githttp.BasicAuth{
						Username: user.String(), // anything except an empty string
						Password: loadToken(),
					},
				})
				if err != nil {
					fmt.Println(err)
				}
			}
			repos[repoSlice[i].Name] = repoSlice[i].HTMLURL
		}
	}
	allRepos["repos"] = repos
	return repos
}

func backupStars(clone bool) map[string]any {

	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	fmt.Println("User: " + user.String())
	var starSlice []repoStruct
	stars := map[string]any{}
	neededPages := calculateNeededPages("userStars")
	fmt.Println("Getting list of starred repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/starred?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &starSlice)
		if err != nil {
			fmt.Printf("Error:\n%v\n", err)
		}
		for i := 0; i < len(starSlice); i++ {
			owner := starSlice[i].Owner.Login
			if clone {
				cloneDirectory := filepath.Dir(currentDirectory + "/github-backup-" + dateToday + "/your-stars/" + owner + "_" + starSlice[i].Name + "/")
				fmt.Printf("Cloning %v (iteration %v) to %v\n", starSlice[i].Name, i, cloneDirectory)
				_, err = git.PlainClone(cloneDirectory, false, &git.CloneOptions{
					URL: starSlice[i].HTMLURL,
					Auth: &githttp.BasicAuth{
						Username: user.String(), // anything except an empty string
						Password: loadToken(),
					},
				})
				if err != nil {
					fmt.Println(err)
				}
			}
			stars[starSlice[i].Name] = starSlice[i].HTMLURL
		}
	}
	allRepos["stars"] = stars
	return stars
}

func calculateNeededPages(whichRepos string) int {
	if whichRepos == "userRepos" {
		response := ghRequest("https://api.github.com/user")
		json := responseContent(response.Body)
		publicRepos := gjson.Get(json, "public_repos")
		privateRepos := gjson.Get(json, "total_private_repos")
		totalRepos := publicRepos.Num + privateRepos.Num
		fmt.Println("Total repositories: " + strconv.Itoa(int(totalRepos)))
		neededPages := math.Ceil(totalRepos / 100)
		// fmt.Println(neededPages)
		fmt.Println("Total pages needed:" + strconv.Itoa(int(neededPages)))
		return int(neededPages)
	} else if whichRepos == "userStars" {
		var starSlice []repoStruct
		pageNumber := 1
		perPage := 100
		starJSON := responseContent(ghRequest("https://api.github.com/user/starred?page=1&per_page=" + strconv.Itoa(perPage)).Body)
		err := json.Unmarshal([]byte(starJSON), &starSlice)
		if err != nil {
			fmt.Printf("Error:\n%v\n", err)
		}
		for len(starSlice) != 0 { // if len(starSlice) == perPage {
			pageNumber++
			starJSON := responseContent(ghRequest("https://api.github.com/user/starred?page=" + strconv.Itoa(pageNumber) + "&per_page=" + strconv.Itoa(perPage)).Body)
			err := json.Unmarshal([]byte(starJSON), &starSlice)
			if err != nil {
				fmt.Printf("Error:\n%v\n", err)
			}

		}
		return pageNumber
	} else {
		return 0
	}
}

func responseContent(responseBody io.ReadCloser) string {
	bytes, err := io.ReadAll(responseBody)
	if err != nil {
		fmt.Printf("Error:\n%v\n", err)
	}
	return string(bytes)
}

func ghRequest(url string) *http.Response {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("Error:\n%v\n", err)
	}
	req.Header.Set("Authorization", "token "+loadToken())
	// req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Accept", "testing-github-api")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error:\n%v\n", err)
	}
	if response.StatusCode != 200 {
		fmt.Println("Something went wrong, status code is not \"200 OK\"")
		fmt.Println("Response status code: " + response.Status)
		if response.StatusCode == 401 {
			fmt.Println("You may have an incorrect Github personal token")
		}
		os.Exit(1)

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
		backupRepos(true)
	case "2":
		backupStars(true)
	case "3":
		backupRepos(true)
		backupStars(true)
	}
}

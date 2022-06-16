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

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/joho/godotenv"
	"github.com/tidwall/gjson"
)

type repoStruct struct {
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

func main() {
	godotenv.Load()
	fmt.Println(`What should be backed up?
1) Your public and private repositories
2) Your starred repositories
3) Both`)
	reader := bufio.NewReader(os.Stdin)
	backupChoice, _ := reader.ReadString('\n')
	backupChoice = strings.TrimSpace(backupChoice)
	switch backupChoice {
	case "1":
		cloneRepos()
	case "2":
		cloneStars()
	case "3":
		cloneRepos()
		cloneStars()
	}
}

func cloneRepos() {
	currentDirectory, _ := os.Getwd()
	dateToday := time.Now().Format("01-02-2006")
	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	fmt.Println("User: " + user.String())
	var repoSlice []repoStruct
	neededPages := calculateNeededPages("userRepos")
	fmt.Println("Getting list of repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/repos?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &repoSlice)
		if err != nil {
			fmt.Printf("Error:\n%v\n", err)
		}
		for i := 0; i < len(repoSlice); i++ {
			cloneDirectory := filepath.Dir(currentDirectory + "/github-backup/your-repos-" + dateToday + "/" + repoSlice[i].Name + "/")
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
	}
}

func cloneStars() {
	currentDirectory, _ := os.Getwd()
	dateToday := time.Now().Format("01-02-2006")
	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	fmt.Println("User: " + user.String())
	var starSlice []repoStruct
	neededPages := calculateNeededPages("userStars")
	fmt.Println("Getting list of starred repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/starred?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &starSlice)
		if err != nil {
			fmt.Printf("Error:\n%v\n", err)
		}
		for i := 0; i < len(starSlice); i++ {
			cloneDirectory := filepath.Dir(currentDirectory + "/github-backup/your-stars-" + dateToday + "/" + starSlice[i].Name + "/")
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
	}
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

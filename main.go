package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	// response := ghRequest("https://api.github.com/user/repos")
	// fmt.Println(responseContent(response.Body))
	cloneRepos()
}

func cloneRepos() {
	currentDirectory, _ := os.Getwd()
	dateToday := time.Now().Format("01-02-2006")
	var repoSlice []repoStruct
	for i := 1; i <= calculateNeededPages(); i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/repos?page" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &repoSlice)
		if err != nil {
			fmt.Printf("Error:\n%v\n", err)
		}
		fmt.Println(repoSlice)
		for i := 0; i < len(repoSlice); i++ {
			cloneDirectory := filepath.Dir(currentDirectory + "/github-backup-" + dateToday + "/" + repoSlice[i].Name + "/")
			user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
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

func calculateNeededPages() int {
	response := ghRequest("https://api.github.com/user")
	json := responseContent(response.Body)
	publicRepos := gjson.Get(json, "public_repos")
	privateRepos := gjson.Get(json, "total_private_repos")
	totalRepos := publicRepos.Num + privateRepos.Num
	neededPages := math.Ceil(totalRepos / 100)
	// fmt.Println(neededPages)
	return int(neededPages)
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
	return response
}

func loadToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

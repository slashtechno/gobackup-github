package main

import (
	"bufio"
	"encoding/json"
	"flag"
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

	"github.com/TwiN/go-color"
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

var (
	currentDirectory, _ = os.Getwd()
	backupDirectory     = filepath.Join(currentDirectory, "github-backup-"+dateToday)
)

var (
	dateToday = time.Now().Format("01-02-2006")
	allRepos  = map[string]any{}
)

var (
	repoFlag           = flag.Bool("backup-repos", false, "Set this flag to backup your repositories and SKIP the interactive UI (can be combined with backup-stars)")
	starFlag           = flag.Bool("backup-stars", false, "Set this flag to backup your starred repositoriesand SKIP the interactive UI (can be combined with backup-repos)")
	createStarListFlag = flag.Bool("create-star-list", false, "Set this flag to create a list of starred repositories")
	createRepoListFlag = flag.Bool("create-repo-list", false, "Set this flag to create a list of your repositories")
	skipCloneFlag      = flag.Bool("skip-clone", false, "Set this flag to skip cloning repositories. Recommended use is when creating records of repositories")
	noInteractFlag     = flag.Bool("skip-interaction", false, "Set this flag to skip the user interface. Recommended use is when creating records of repositories")
	ranRepoBackup      = false
	ranStarBackup      = false
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), color.InBold("Usage:"))
		flag.PrintDefaults()
	}
	godotenv.Load()
	flag.Parse()
	if *repoFlag && *starFlag {
		backupRepos(!*skipCloneFlag)
		backupStars(!*skipCloneFlag)
	} else if *repoFlag {
		backupRepos(!*skipCloneFlag)
	} else if *starFlag {
		backupStars(!*skipCloneFlag)
	} else if *noInteractFlag {
		log.Println("\"skip-interaction\" was set. Skipping UI")
	} else {
		mainMenu()
	}
	savedRepos := make(map[string]any, 2)
	// for k, v := range allRepos {
	// 	savedRepos[k] = v
	// }
	if *createRepoListFlag {
		if !ranRepoBackup {
			backupRepos(false)
		}
		savedRepos["repos"] = allRepos["repos"]
	}
	if *createStarListFlag {
		if !ranStarBackup {
			backupStars(false)
		}
		savedRepos["stars"] = allRepos["stars"]
	}
	if len(savedRepos) != 0 {
		_, err := os.Stat(backupDirectory)
		if os.IsNotExist(err) {
			file, err := os.Create(filepath.Join(currentDirectory, "github-repository-list-") + dateToday + ".json")
			log.Println("Creating file at " + filepath.Join(currentDirectory, "github-repository-list-") + dateToday + ".json") // For debugging
			checkNilErr(err)
			repoJsonByte, err := json.MarshalIndent(savedRepos, "", "    ")
			checkNilErr(err)
			file.Write(repoJsonByte)
		} else {
			file, err := os.Create(filepath.Join(backupDirectory, "github-repository-list.json"))
			log.Println("Creating file at " + filepath.Join(backupDirectory, "github-repository-list.json")) // For debugging
			checkNilErr(err)
			repoJsonByte, err := json.MarshalIndent(savedRepos, "", "    ")
			checkNilErr(err)
			file.Write(repoJsonByte)
		}
	} else {
		log.Println("No repositories were recorded")
	}
}

func backupRepos(clone bool) map[string]any {
	ranRepoBackup = true
	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	log.Println("User: " + user.String())
	var repoSlice []repoStruct
	repos := map[string]any{}
	neededPages := calculateNeededPages("userRepos")
	log.Println("Getting list of repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/repos?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &repoSlice)
		checkNilErr(err)
		for i := 0; i < len(repoSlice); i++ {
			owner := repoSlice[i].Owner.Login
			if clone {
				cloneDirectory := filepath.Join(backupDirectory, "your-repos", owner+"_"+repoSlice[i].Name)
				log.Printf("Cloning %v (iteration %v) to %v\n", repoSlice[i].Name, i+1, cloneDirectory)
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
	allRepos["repos"] = repos
	return repos
}

func backupStars(clone bool) map[string]any {
	ranStarBackup = true
	user := gjson.Get(responseContent(ghRequest("https://api.github.com/user").Body), "login")
	log.Println("User: " + user.String())
	var starSlice []repoStruct
	stars := map[string]any{}
	neededPages := calculateNeededPages("userStars")
	log.Println("Getting list of starred repositories")
	for i := 1; i <= neededPages; i++ {
		repoJSON := responseContent(ghRequest("https://api.github.com/user/starred?per_page=100&page=" + strconv.Itoa(i)).Body)
		err := json.Unmarshal([]byte(repoJSON), &starSlice)
		checkNilErr(err)
		for i := 0; i < len(starSlice); i++ {
			owner := starSlice[i].Owner.Login
			if clone {
				cloneDirectory := filepath.Join(backupDirectory, "your-stars", owner+"_"+starSlice[i].Name)
				log.Printf("Cloning %v (iteration %v) to %v\n", starSlice[i].Name, i, cloneDirectory)
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
		log.Println("Total repositories: " + strconv.Itoa(int(totalRepos)))
		neededPages := math.Ceil(totalRepos / 100)
		// fmt.Println(neededPages)
		log.Println("Total pages needed:" + strconv.Itoa(int(neededPages)))
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
			log.Println("You may have an incorrect Github personal token")
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
	flag.Parse()
	fmt.Println(`What should this program backup?
1) Your public and private repositories
2) Your starred repositories
3) Both`)
	reader := bufio.NewReader(os.Stdin)
	backupSelection, _ := reader.ReadString('\n')
	backupSelection = strings.TrimSpace(backupSelection)
	switch backupSelection {
	case "1":
		backupRepos(!*skipCloneFlag)
	case "2":
		backupStars(!*skipCloneFlag)
	case "3":
		backupRepos(!*skipCloneFlag)
		backupStars(!*skipCloneFlag)
	}
}

func checkNilErr(err any) {
	if err != nil {
		// log.Fatalln("Error:\n%v\n", err)
		log.Fatalln(err)
	}
}

# gobackup-github  
Go program that utilizes the Github API to backup all repositories for a user  

### Setup  
1. Create a Github personal access token with the following scopes:  `read:user, repo`  
2. Either download a binary for your system from releases, or build the program yourself with `go build` (`go run` can also work)  
3. Create a `.env` file  
4. Add `GITHUB_TOKEN = TOKEN` to `.env` (replace TOKEN with your token)  
5. Run the binary in the same download as your `.env` file  

### Why?  
I wanted a simple way to backup my Github repositories. I also wanted to learn Go and APIs. Thus, I started this project as a way to create my first Go project, use the Github API, and make a utility to backup Github repositories.  
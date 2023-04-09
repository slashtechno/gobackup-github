# gobackup-github  
[![Build on multiple platforms](https://github.com/slashtechno/gobackup-github/actions/workflows/go-build.yml/badge.svg)](https://github.com/slashtechno/gobackup-github/actions/workflows/go-build.yml)![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/slashtechno/gobackup-github)  
Go program that utilizes the Github API to backup all repositories for a user. In addition, can backup starred repositories  

### Setup  
1. Create a Github personal access token with the following scopes:  `read:user, repo`  
2. Either download a binary for your system from releases, or build the program yourself with `go build`   
3. Create a `.env` file  
4. Add `GITHUB_TOKEN = TOKEN` to `.env` (replace TOKEN with your token)  
5. Run the binary in the same folder as your `.env` file  

### CLI Examples  
These are examples, for full usage info, run `gobackup-github --help`  
* `gobackup-github` - Use the interactive UI 
* `gobackup-github backup repos` - Backup your repositories 
* `gobackup-github backup gists` - Backup your gists 
* `gobackup-github backup stars` - Backup your starred repositories 
* `gobackup-github repos stars` - Backup your repositories and your starred repositories  
* `gobackup-github backup --create-list --no-clone repos stars` - Record the names and URLs of starred and owned repositories, without cloning them  

### Why?  
I wanted a simple way to backup my Github repositories. I also wanted to learn Go and APIs. Thus, I started this project as a way to create my first Go project, use the Github API, and make a utility to backup Github repositories.  

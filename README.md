# gobackup-github  
[![Build on multiple platforms](https://github.com/slashtechno/gobackup-github/actions/workflows/go-build.yml/badge.svg)](https://github.com/slashtechno/gobackup-github/actions/workflows/go-build.yml)  
Go program that utilizes the Github API to backup all repositories for a user. In addition, can backup starred repositories  

### Setup  
1. Create a Github personal access token with the following scopes:  `read:user, repo`  
2. Either download a binary for your system from releases, or build the program yourself with `go build` (`go run` can also work)  
3. Create a `.env` file  
4. Add `GITHUB_TOKEN = TOKEN` to `.env` (replace TOKEN with your token)  
5. Run the binary in the same download as your `.env` file  

### CLI Flag Examples  
These are examples, for full usage info, run `gobackup-github -h`  
* `gobackup-github` - Use the interactive UI (flags can be added)  
* `gobackup-github --backup-repos` - Backup your repositories without using the UI  
* `gobackup-github --backup-stars` - Backup your starred repositories without using the UI  
* `gobackup-github --backup-stars --backup-repos` - Backup your repositories and your starred repositories  
* `gobackup-github --list-only` - Record the names and URLs of repositories selected in the UI in a JSON file (skip cloning)    
* `gobackup-github --backup-repos --skip-list` - Backup your repositories and skip creating a JSON file  

### Why?  
I wanted a simple way to backup my Github repositories. I also wanted to learn Go and APIs. Thus, I started this project as a way to create my first Go project, use the Github API, and make a utility to backup Github repositories.

### Roadmap  
- [ ] Simplify command line flags
- [ ] Allow interactive backups of individual starred repositories

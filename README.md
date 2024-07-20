# gobackup-github  
[![Build on multiple platforms](https://github.com/slashtechno/gobackup-github/actions/workflows/go-build.yml/badge.svg)](https://github.com/slashtechno/gobackup-github/actions/workflows/go-build.yml)![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/slashtechno/gobackup-github)[![Create and publish a Docker image](https://github.com/slashtechno/gobackup-github/actions/workflows/docker.yml/badge.svg?branch=main)](https://github.com/slashtechno/gobackup-github/actions/workflows/docker.yml)  
Go program that utilizes the Github API to backup all a user's repositories, including repositories that have been stored. Multiple users can be backed up, including all members of a GitHub organization.

### Setup  
1. Create a Github personal access token with the following scopes:  `read:user, repo`  
2. Either download a binary for your system from releases, or build the program and add it to your PATH with `go install`.
3. Copy `config.example.yaml` to `config.yaml` and fill in the required fields
4. Run the program with `gobackup-github backup`  

### Docker  
This program can also be run in Docker.  
To pull the image, run `docker pull ghcr.io/slashtechno/gobackup-github:latest`  
To create a container that removes itself after, run `docker run --rm -it -v ${PWD}/config.yaml:/config.yaml -v ${PWD}/backup:/backups ghcr.io/slashtechno/gobackup-github:latest`

### Why?  
I wanted a simple way to backup my Github repositories. I also wanted to learn Go and APIs. Thus, I began this project in 2022 as a way to create my first Go project, use the Github API, and make a utility to backup Github repositories.  
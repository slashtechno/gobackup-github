# gobackup-github  
[![GitHub Actions Build Workflow Status](https://img.shields.io/github/actions/workflow/status/slashtechno/gobackup-github/go-build.yml?style=for-the-badge&label=Build&labelColor=%2344cc11&color=%23555555)](https://github.com/slashtechno/gobackup-github/actions/workflows/go-build.yml) ![Static Badge](https://img.shields.io/badge/open-source-_?style=for-the-badge&labelColor=%23ef4041&color=%23c13a3a) [![GitHub Actions Docker Workflow Status](https://img.shields.io/github/actions/workflow/status/slashtechno/gobackup-github/docker.yml?style=for-the-badge&label=Docker%20Image%20Build)](https://github.com/slashtechno/gobackup-github/actions/workflows/docker.yml)

Go program that utilizes the Github API to backup a user's repositories, including repositories that have been starred. Multiple users can be backed up, including all members of a GitHub organization.
![Demo](demo.gif)

### Setup and Usage
1. Create a Github personal access token with the following scopes:  `read:user, repo`  
2. Either download a binary for your system from releases, or build the program and add it to your PATH with `go install`.
3. Copy `config.example.yaml` to `config.yaml` and fill in the fields.
    - In addition, command line flags can be used to specify configuration options. Use the `help` command or the `--help` flag for more information.
    - It is recommended to read through the `config.example.yaml` file to understand the configuration options.
4. Run the program with `gobackup-github backup` 
    - To perform a rolling backup, run `gobackup-github backup continuous`

### Docker  
This program can also be run in Docker.  
<!-- To pull the image, run `docker pull ghcr.io/slashtechno/gobackup-github:latest`   -->
To create a container that removes itself after exiting, run `docker run --rm -it -v ${PWD}/config.yaml:/config.yaml -v ${PWD}/backup:/backups ghcr.io/slashtechno/gobackup-github:latest`  
To create a container that runs on boot and performs a rolling backup every 24 hours, run `docker run --restart unless-stopped -d --name gobackup-github -v ${PWD}/config.yaml:/config.yaml -v ${PWD}/backup:/backups ghcr.io/slashtechno/gobackup-github:latest continuous -i 24h`

#### Docker Compose  
You can also just run `docker compose up -d` to start the container with automatic restarts, assuming you have a `docker-compose.yml` file in the same directory as the `config.yaml` file. You can also edit the `docker-compose.yml` file to change configuration and to manage the rolling backup, if needed.

### Contributing  
Pull Requests are welcome!  
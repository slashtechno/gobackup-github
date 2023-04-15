FROM golang:1.20.2-bullseye
WORKDIR /go/src/github.com/slashtechno/gobackup-github
COPY . ./
RUN go install

# Create the script
RUN echo "#!/bin/bash" > /usr/local/bin/backup
RUN echo "gobackup-github backup \"\$@\"" >> /usr/local/bin/backup

# Script Contents ($@ is all arguments passed to the script):
# #!/bin/bash
# longcommand "$@"

# Make the script executable
RUN chmod +x /usr/local/bin/backup

# CMD ["gobackup-github"]

# Build: docker build -t gobackup-github .
# Single run:
# docker run --rm -it --name gobackup-github -v ${PWD}/backups:/backups --env-file ${PWD}/.env gobackup-github backup -d /backups -c repos gists
# Repeating run:
# docker run -it -d --name gobackup-github -v ${PWD}/backups:/backups --env-file ${PWD}/.env gobackup-github backup /backups -r -c repos gists
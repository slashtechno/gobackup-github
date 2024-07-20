FROM golang:1.22
WORKDIR /app
COPY . .
RUN go install
ENTRYPOINT ["gobackup-github", "--config", "/config.yaml", "backup", "--output", "/backups"]
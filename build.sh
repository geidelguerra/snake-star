#!/usr/bin/env bash
# Build script for Snake Star project
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o ./bin/snake_star_linux_amd64 main.go

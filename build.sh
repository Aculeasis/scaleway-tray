#!/bin/bash
GIT_COMMIT="$(git rev-list -1 HEAD)"
VERSION="$(git describe --always --abbrev=0 --tags)"
BUILD_DATE="$(date -u '+%d.%m.%y %H:%M:%S UTC')"

go build -ldflags="-X main.GitCommit=$GIT_COMMIT -X 'main.BuildDate=$BUILD_DATE' -X main.Version=$VERSION -s -w" -o ./bin/scaleway-tray ./src

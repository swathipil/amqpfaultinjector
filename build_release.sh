#!/bin/bash
set -ex

VERSION=$1

if [ -z "$VERSION" ]; then
  echo "Version not provided"
  exit 1
fi

GIT_ROOT=$(git rev-parse --show-toplevel)
pushd $GIT_ROOT
rm -rf _bin
mkdir -p _bin/windows
mkdir -p _bin/linux

# build go binaries for stuff in cmd
GOOS=windows GOARCH=amd64 go build -o _bin/windows ./cmd/...

# also for linux
GOOS=linux GOARCH=amd64 go build -o _bin/linux ./cmd/...

git fetch origin main
git tag "$VERSION" origin/main
git push origin "$VERSION"

# create and upload github release
gh release create $VERSION _bin/windows/* _bin/linux/*

#!/usr/bin/env bash

for tool in glide; do
    which "$tool" &>/dev/null && continue
    case $tool in
        glide)
            go get -v github.com/Masterminds/glide
            (cd "$GOPATH/src/github.com/Masterminds/glide" && git checkout tags/v0.12.3 && go install);;
        *) echo "Don't know how to install $tool"; exit 1;;
    esac
done

glide i

mkdir -p rackn-sledgehammer

go build -o rackn-sledgehammer/gohai gohai/main.go


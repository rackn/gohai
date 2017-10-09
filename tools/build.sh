#!/usr/bin/env bash

glide i

mkdir -p rackn-sledgehammer

go build -o rackn-sledgehammer/gohai gohai/main.go


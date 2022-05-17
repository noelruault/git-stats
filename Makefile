#!/usr/bin/make -f

.ONESHELL:
.SHELL := /usr/bin/bash


help:
	echo "Usage: make [options] [arguments]\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

install:
	yarn install

build:
	go build -o bin/git-stats

run: build
	./bin/git-stats -test=false # -gitlab.token="" -github.token="" -gitlab.user=""
	npx node-html-to-image-cli out/charts/lines.html out/images/lines.png

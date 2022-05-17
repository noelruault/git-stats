#!/usr/bin/make -f

.ONESHELL:
.SHELL := /usr/bin/bash


help:
	@echo "Usage: make [options] [arguments]\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

install: ## Install dependecies
	yarn install

build: ## Build golang project
	go build -o bin/git-stats

run: build ## Build and run the project
	./bin/git-stats -test=false -github.token="" -gitlab.token="" -gitlab.user=""
	npx node-html-to-image-cli out/charts/lines.html out/images/lines.png

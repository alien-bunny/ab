#!/bin/bash

if [ -e .env ]; then
	echo "Using .env"
	set -o allexport
	source ./.env
	set +o allexport
fi

go test -race -p $(nproc) -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic ./...

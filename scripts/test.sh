#!/bin/bash

if [ -e .env ]; then
	echo "Using .env"
	set -o allexport
	source ./.env
	set +o allexport
fi

ginkgo ${@} -r -p --failOnPending --cover --trace --progress

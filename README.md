# Alien Bunny

[![Build Status](https://travis-ci.org/alien-bunny/ab.svg?branch=v1)](https://travis-ci.org/alien-bunny/ab)
[![codecov](https://codecov.io/gh/alien-bunny/ab/branch/v1/graph/badge.svg)](https://codecov.io/gh/alien-bunny/ab)
[![Go Report Card](https://goreportcard.com/badge/github.com/alien-bunny/ab)](https://goreportcard.com/report/github.com/alien-bunny/ab)
[![CodeFactor](https://www.codefactor.io/repository/github/alien-bunny/ab/badge)](https://www.codefactor.io/repository/github/alien-bunny/ab)

Alien Bunny is a content management framework, written in Go.

## Getting started

Quick install (without server side JavaScript support):

	go get -d -u github.com/alien-bunny/ab
	cd $GOPATH/src/github.com/alien-bunny/ab/cmd/abt
	go install

To install the frontend:

	npm install --save abjs

Alternatively, if you want to work on the development version, use [npm link](https://docs.npmjs.com/cli/link):

	cd $GOPATH/github.com/alien-bunny/ab/js
	npm link
	cd $YOUR_PROJECT
	npm link abjs

See examples and more information at [the project website](http://www.alien-bunny.org).

## Requirements

* Go 1.11
* PostgreSQL 10 or newer.
* Frontend components and the scaffolded application base require NPM 3+.

### Database requirements:

The `uuid` extension must be installed on the database.

    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

## The abt command

The `abt` command is a helper tool for the development. Subcommands:

* `watch`: starts a proxy that builds and reruns your application when you save a file.
* `gensecret`: generates a secret key. Useful for generating cookie secrets.
* `decrypt`: decrypts a string with the util package

## Testing

An evironment variable called `AB_TEST_DB` must be defined with the connection string to the PostgreSQL database.

## Contributing

Feel free to open an issue / pull request for discussion, feature request or bug report. If you plan to make a bigger pull request, open an issue before you start implementing it.

## Development

### Release criteria

* No critical issues
* High test coverage: 90% is a good target, but the point is to cover all non-trivial code paths
* Clean go vet
* All exported functions/types have documentation
* No IDEA/Goland inspection problems
* No TODOs

Major releases only:

* No major issues 
* Migration guide

### Breaking changes

What is a breaking change:

* Public Go API
* Configuration format

What isn't a breaking change:

* Go, PostgreSQL, NPM minimum version
* Dependency versions
* TLS-related requirements (curves, ciphers etc)
* Default crypto hash algorithm (e.g. password hash)
* Default HMAC algorithm (e.g. session signature)
* Default encryption algorithm (e.g. authentication data)
* Internal list additions (e.g. list of decoders, list of disabled account names)

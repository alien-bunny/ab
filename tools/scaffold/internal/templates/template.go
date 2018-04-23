// Copyright 2018 Tamás Demeter-Haludka
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"os"
	"strings"
	"text/template"

	"github.com/alien-bunny/ab/lib/util"
)

func ScaffoldApp(name string) error {
	if err := os.Mkdir(name, 0755); err != nil {
		return err
	}

	if err := os.Chdir(name); err != nil {
		return err
	}
	defer os.Chdir("..")

	if err := ensureDirs(appDirs); err != nil {
		return err
	}

	appdata := AppData{
		Config: ConfigData{
			Secret:       util.RandomSecret(32),
			CookieSecret: util.RandomSecret(32),
			Host:         "localhost",
			Port:         "8080",
		},
		Index: IndexData{
			Title: nameToTitle(name),
		},
	}

	simpleAppTemplates := []*template.Template{
		webpackconf,
		packagejson,
		bootstrapconfigjs,
		bootstrapconfigless,
		styleless,
		humanstxt,
		robotstxt,
		gitignore,
	}

	for _, t := range simpleAppTemplates {
		if err := renderToFile(t, appdata); err != nil {
			return err
		}
	}

	return nil
}

type AppData struct {
	Config ConfigData
	Index  IndexData
}

func nameToTitle(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Title(name)
}

var appDirs = []string{
	"assets",
	"cmd/server",
	"fonts",
	"html",
	"images",
	"js/action",
	"js/build",
	"js/component/stateful",
	"js/component/stateless",
	"js/middleware",
	"js/reducer",
	"less",
	"lib/server",
	"private",
	"public",
	"worker",
}

func ensureDirs(dirs []string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func renderToFile(t *template.Template, data interface{}) error {
	file, err := os.Open(t.Name())
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, data)
}

var webpackconf = template.Must(template.New("webpack.config.js").Parse(`"use strict";
const base = require("./node_modules/abjs/webpack.base.js");

base.targets.forEach(function (target) {
});

module.exports = base.targets;
`))

var packagejson = template.Must(template.New("package.json").Parse(`{
	"scripts": {
		"build": "webpack --progress --profile --colors",
		"watch": "webpack --progress --profile --colors --watch",
		"debug": "webpack --progress --profile --colors --display-reasons --display-error-details --display-modules",
		"check": "npm-check -s",
		"update": "npm-check -u"
	},
	"dependencies": {
		"abjs": "*",
		"babel-plugin-dev-expression": "^0.2.1",
		"babel-plugin-object-rest-spread": "0.0.0",
		"babel-plugin-transform-class-properties": "^6.11.5",
		"babel-plugin-transform-export-extensions": "^6.8.0",
		"babel-plugin-transform-function-bind": "^6.8.0",
		"babel-plugin-transform-object-assign": "^6.8.0",
		"babel-plugin-transform-object-rest-spread": "^6.8.0",
		"babel-polyfill": "^6.13.0",
		"babel-preset-es2015": "^6.14.0",
		"babel-preset-react": "^6.11.1",
		"phaser": "^2.6.2",
		"webpack": "^1.13.2"
	},
	"devDependencies": {
		"npm-check": "^5.2.3"
	}
}
`))

var bootstrapconfigjs = template.Must(template.New("bootstrap.config.js").Parse(`"use strict";

module.exports = require("abjs/bootstrap.config.js");
`))

var bootstrapconfigless = template.Must(template.New("bootstrap.config.less").Parse(`@import "less/style.less";
`))

var styleless = template.Must(template.New("less/style.less").Parse(``))

type ConfigData struct {
	Secret       string
	CookieSecret string
	Host         string
	Port         string
}

var configjson = template.Must(template.New("config.json").Parse(`{
	"db": "",
	"loglevel": 2,
	"secret": "{{.Secret}}",
	"cookiesecret": "{{.CookieSecret}}",
	"baseurl": "http://{{.Host}}:{{.Port}}/",
	"host": "{{.Host}}"
	"port": "{{.Port}}"
}
`))

var humanstxt = template.Must(template.New("humans.txt").Parse(``))

var robotstxt = template.Must(template.New("robots.txt").Parse(`User-Agent: *
Disallow: /api
`))

type IndexData struct {
	Title string
}

var indexhtml = template.Must(template.New("html/index.html").Parse(`<!DOCTYPE HTML>
<html class="no-js" lang="en">
	<head>
		<meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1" />
		<meta charset="utf8" />
		<meta name="viewport" content="width=device-width, minimum-scale=1.0, initial-scale=1.0, user-scalable=yes" />
		<title>{{.Title}}</title>
		<link type="text/plain" rel="author" href="/humans.txt" />
		<script type="text/javascript">
			try {
				window.INITIAL_STATE = JSON.parse("{{"{{"}}.InitialState{{"}}"}}");
			} catch (ex) {}
		</script>
	</head>
	<body>
		<div id="app">{{"{{"}}.Content{{"}}"}}</div>
	</body>
</html>
`))

var gitignore = template.Must(template.New(".gitignore").Parse(`node_modules
assets
worker
config.json
`))

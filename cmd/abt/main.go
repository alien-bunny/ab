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

package main

import (
	"os"

	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/tools/decrypt"
	"github.com/alien-bunny/ab/tools/gensecret"
	"github.com/alien-bunny/ab/tools/scaffold"
	"github.com/alien-bunny/ab/tools/session"
	"github.com/alien-bunny/ab/tools/version"
	"github.com/spf13/cobra"
)

func main() {
	logger := log.NewProdLogger(os.Stdout)

	abtCmd := &cobra.Command{
		Use:   "abt",
		Short: "abt is a command line helper for Alien Bunny",
	}

	abtCmd.AddCommand(
		gensecretcmd.CreateGenSecretCMD(logger),
		decryptcmd.CreateDecryptCMD(logger),
		sessioncmd.CreateSessionCMD(logger),
		scaffoldcmd.CreateScaffoldCMD(logger),
		versioncmd.CreateVersionCMD(logger),
	)

	abtCmd.Execute()
}

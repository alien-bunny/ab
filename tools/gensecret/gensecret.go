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

package gensecretcmd

import (
	"fmt"

	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/spf13/cobra"
)

func CreateGenSecretCMD(logger log.Logger) *cobra.Command {
	gscmd := &cobra.Command{
		Use:   "generate-secret",
		Short: "generates a secret value",
	}

	length := gscmd.Flags().Int("length", 32, "length of the secret value")

	gscmd.Run = func(c *cobra.Command, args []string) {
		fmt.Println(util.RandomSecret(*length))
	}

	return gscmd
}

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

package versioncmd

import (
	"fmt"

	"github.com/alien-bunny/ab"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/spf13/cobra"
)

func CreateVersionCMD(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "prints version",
	}

	cmd.RunE = func(c *cobra.Command, args []string) error {
		fmt.Println(ab.VERSION)
		return nil
	}

	return cmd
}

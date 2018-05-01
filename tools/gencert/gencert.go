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

package gencert

import (
	"fmt"

	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/spf13/cobra"
)

func CreateGencertCMD(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-cert",
		Short: "Generates a simple cerficiate",
		Args:  cobra.ExactArgs(2),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		host, org := args[0], args[1]
		cert, key := util.GenerateCertificate(host, org)

		fmt.Printf("%s\n\n%s\n", cert, key)

		return nil
	}

	return cmd
}

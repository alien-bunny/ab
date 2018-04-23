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

package sessioncmd

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/session"
	"github.com/spf13/cobra"
)

func CreateSessionCMD(logger log.Logger) *cobra.Command {
	scmd := &cobra.Command{
		Use:   "session",
		Short: "session-related commands",
	}

	decode := &cobra.Command{
		Use:   "decode",
		Short: "dumps and verifies a session",
	}

	decode.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("first argument is the encoded session, second is the key (optional)")
		}

		encoded := args[0]
		var key session.SecretKey = nil
		if len(args) > 1 {
			decoded, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			key = session.SecretKey(decoded)
		}

		sess, err := session.DecodeSession(encoded, key)
		if err != nil {
			return err
		}

		for k, v := range sess {
			fmt.Println(k + "\t" + v)
		}

		return nil
	}

	encode := &cobra.Command{
		Use:   "encode",
		Short: "encodes and signs a flat JSON into a session",
	}

	encode.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("first argument is the map in JSON format, the second is the key")
		}

		var data session.Session
		if err := json.Unmarshal([]byte(args[0]), &data); err != nil {
			return err
		}

		decoded, err := hex.DecodeString(args[1])
		if err != nil {
			return err
		}
		key := session.SecretKey(decoded)

		fmt.Println(session.EncodeSession(data, key))

		return nil
	}

	scmd.AddCommand(
		decode,
		encode,
	)

	return scmd
}

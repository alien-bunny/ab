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

package util_test

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/alien-bunny/ab/lib/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Describe("A random key", func() {
		key := make([]byte, 32)
		io.ReadFull(rand.Reader, key)
		c, err := util.CreateCipher(key)
		It("should be set without an error", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
		})

		Describe("A secret message", func() {
			It("should be encrypted and decrypted", func() {
				rawmsg := make([]byte, 4096)
				_, err := io.ReadFull(rand.Reader, rawmsg)
				Expect(err).NotTo(HaveOccurred())
				msg := hex.EncodeToString(rawmsg)

				encrypted := util.EncryptString(c, msg)
				Expect(encrypted).NotTo(BeZero())

				decrypted, err := util.DecryptString(c, encrypted)
				Expect(err).NotTo(HaveOccurred())
				Expect(decrypted).NotTo(BeZero())

				Expect(decrypted).To(Equal(msg))
			})
		})
	})

	DescribeTable("Placeholder intervals",
		func(start, end int, placeholders string, expected bool) {
			result := util.GeneratePlaceholders(uint(start), uint(end))
			if expected {
				Expect(placeholders).To(Equal(result))
			} else {
				Expect(placeholders).NotTo(Equal(result))
			}
		},
		Entry("1->1", 1, 1, "", true),
		Entry("1->2", 1, 2, "$1", true),
		Entry("1->5", 1, 5, "$1, $2, $3, $4", true),
		Entry("2->5", 2, 5, "$2, $3, $4", true),
		Entry("2->3", 2, 3, "$2, $3", false),
	)
})

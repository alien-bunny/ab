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

package cryptmw_test

import (
	"crypto/rand"
	"net/http"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/cryptmw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Crypt middleware", func() {
	buf := make([]byte, 32)
	rand.Read(buf)

	logger := abtest.GetLogger()
	cmw, err := cryptmw.NewCryptMiddleware(buf)

	if err != nil {
		panic(err)
	}

	stack := middleware.NewStack(nil)
	stack.Push(logmw.New(logger))
	stack.Push(cmw)

	It("should encrypt and decrypt a string", func() {
		msg := util.RandomSecret(1024)
		encrypted := ""

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			encrypted = cryptmw.EncryptString(r, msg)
			Expect(encrypted).NotTo(BeEmpty())
		})
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			decrypted := cryptmw.DecryptString(r, encrypted)
			Expect(decrypted).NotTo(BeEmpty())
			Expect(decrypted).To(Equal(msg))
		})
	})

	It("should encrypt and decrypt a data", func() {
		msg := make([]byte, 4096)
		rand.Read(msg)
		var encrypted []byte

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			encrypted = cryptmw.Encrypt(r, msg)
			Expect(encrypted).NotTo(BeEmpty())
		})
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			decrypted := cryptmw.Decrypt(r, encrypted)
			Expect(decrypted).NotTo(BeEmpty())
			Expect(decrypted).To(Equal(msg))
		})
	})
})

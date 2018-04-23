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

package delaymw_test

import (
	"net/http"
	"time"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/middlewares/delaymw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delay Middleware", func() {
	blockTime := time.Second
	stack := middleware.NewStack(nil)
	stack.Push(delaymw.New(blockTime))

	It("should delay the execution of the middleware chain", func() {
		start := time.Now()
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {})
		duration := time.Since(start)

		Expect(duration).To(BeNumerically(">=", time.Second))
		Expect(duration).To(BeNumerically("<", time.Second+10*time.Millisecond))
	})

})

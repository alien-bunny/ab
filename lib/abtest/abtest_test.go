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

package abtest_test

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/abtest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ABTesting", func() {
	testCases("mock", mockClientFactory)
	testCases("HTTP", clientFactory)
})

func testCases(clientType string, factory func() *abtest.TestClient) {
	Describe("Creating a test "+clientType+" client", func() {
		It("should fetch and assert content", func() {
			By("Creating a test client")
			tc := factory()
			Expect(tc).NotTo(BeNil())

			By("Verifying that the test client asserts the test data")
			tc.Request("POST", "/api/posttest", nil, nil, func(resp *http.Response) {
				td := testdata{}
				tc.AssertJSON(resp, &td, Equal(&posttest))
			}, http.StatusOK)
		})
	})
}

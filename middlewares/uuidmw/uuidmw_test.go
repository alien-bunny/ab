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

package uuidmw_test

import (
	"net/http"

	"github.com/alien-bunny/ab/lib/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UUID middleware", func() {
	validUUID := uuid.Generate(key).String()
	nilUUID := uuid.Nil.String()

	It("should accept a properly signed UUID", func() {
		c := clientFactory()
		c.Request("GET", "/test/"+validUUID, nil, nil, func(resp *http.Response) {
			Expect(c.ReadBody(resp, false)).To(Equal(validUUID))
		}, http.StatusOK)
	})

	It("should reject a bad uuid", func() {
		c := clientFactory()
		c.Request("GET", "/test/"+nilUUID, nil, nil, nil, http.StatusNotFound)
	})

	It("should allow an empty parameter when not strict", func() {
		c := clientFactory()
		c.Request("GET", "/notstrict", nil, nil, nil, http.StatusNoContent)
	})

	It("should not allow an empty parameter when strict", func() {
		c := clientFactory()
		c.Request("GET", "/strict", nil, nil, nil, http.StatusNotFound)
	})
})

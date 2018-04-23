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

package hash_test

import (
	"github.com/alien-bunny/ab/lib/hash"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hash", func() {
	It("should match passwords", func() {
		h, herr := hash.DefaultHashPassword(pw)
		Expect(herr).NotTo(HaveOccurred())
		Expect(h).NotTo(BeZero())

		ok, verr := hash.VerifyPassword(pw, h)
		Expect(verr).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})
})

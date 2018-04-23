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

package uuid_test

import (
	"crypto/rand"

	"github.com/alien-bunny/ab/lib/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gouuid "github.com/satori/go.uuid"
)

var _ = Describe("UUID", func() {
	key := make([]byte, 64)
	rand.Read(key)

	gen := uuid.Generate(key)
	It("should verify itself", func() {
		Expect(gen).NotTo(BeZero())
		Expect(gen.Verify(key)).To(BeTrue())
	})

	It("should match the standard", func() {
		Expect(gen.Version()).To(Equal(uint8(4)))
		Expect(gen.Variant()).To(Equal(uint8(gouuid.VariantRFC4122)))
	})

	gen2 := uuid.Generate(key)
	gen2[0] = 0
	It("should fail the verification if tampered", func() {
		Expect(gen2.Verify(key)).To(BeFalse())
	})

	gen3 := uuid.Generate(key)
	It("should parse and validate", func() {
		Expect(gen3).NotTo(BeZero())
		Expect(uuid.ParseAndVerify(key, gen3.String())).To(BeTrue())
	})

	It("should not parse and validate bad input", func() {
		Expect(uuid.ParseAndVerify(key, "")).To(BeFalse())
	})
})

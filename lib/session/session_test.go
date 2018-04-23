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

package session_test

import (
	"encoding/hex"
	"strings"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/session"
	"github.com/alien-bunny/ab/lib/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	key = session.SecretKey(abtest.FakeKey)
)

func randomSession() session.Session {
	sess := make(session.Session)
	for i := 0; i < 16; i++ {
		sess[util.RandomString(8)] = util.RandomString(16)
	}
	return sess
}

func resign(encoded string) string {
	binary, herr := hex.DecodeString(encoded)
	if herr != nil {
		panic(herr)
	}

	signature := key.Sign(binary[32+1:])

	resigned := append(signature, binary[32:]...)

	return hex.EncodeToString(resigned)
}

var _ = Describe("Session", func() {
	Describe("A session object", func() {
		sess := randomSession()

		It("should encode and decode", func() {
			By("encoding the session")
			encoded := session.EncodeSession(sess, key)

			By("decoding the session")
			decoded, derr := session.DecodeSession(encoded, key)
			Expect(derr).NotTo(HaveOccurred())
			Expect(decoded).To(Equal(sess))
		})

		It("should generate a session id", func() {
			id := sess.Id()
			Expect(id).NotTo(BeZero())
			Expect(id).To(Equal(sess.Id()))
		})
	})

	Describe("A session object, containing 0 byte key or a value", func() {
		sessk := make(session.Session)
		sessv := make(session.Session)

		sessk["asdf\x00qwerty"] = util.RandomString(8)
		sessv[util.RandomString(8)] = "zxcvbn\x00"

		It("should panic when there is a 0 byte in the key", func() {
			Expect(func() {
				session.EncodeSession(sessk, key)
			}).To(Panic())
		})

		It("should panic when there is a 0 byte in the value", func() {
			Expect(func() {
				session.EncodeSession(sessv, key)
			}).To(Panic())
		})
	})

	Describe("An encoded session with an invalid signature", func() {
		sess := randomSession()
		encoded := session.EncodeSession(sess, key)
		invalidSignature := strings.Repeat("0", 64) + encoded[64:]
		tampered := encoded + "00"

		It("should fail with a signature verification error", func() {
			_, derr0 := session.DecodeSession(invalidSignature, key)
			Expect(derr0).To(HaveOccurred())
			Expect(derr0).To(Equal(session.SignatureVerificationFailedError))

			_, derr1 := session.DecodeSession(tampered, key)
			Expect(derr1).To(HaveOccurred())
			Expect(derr1).To(Equal(session.SignatureVerificationFailedError))
		})

		It("should fail when the signature is truncated", func() {
			_, derr := session.DecodeSession(strings.Repeat("0", 64), key)
			Expect(derr).To(HaveOccurred())
			Expect(derr).To(Equal(session.MalformedSessionDataError))
		})
	})

	Describe("A malformed session data", func() {
		malf := session.EncodeSession(randomSession(), key)
		malf0 := resign(malf + "00")
		malf1 := malf + "p0"

		It("should fail when the number of entries are odd", func() {
			_, derr := session.DecodeSession(malf0, key)
			Expect(derr).To(HaveOccurred())
			Expect(derr).To(Equal(session.MalformedSessionDataError))
		})

		It("should fail when the hex decoding fails", func() {
			_, derr := session.DecodeSession(malf1, key)
			Expect(derr).To(HaveOccurred())
		})
	})

	Describe("A SecretKey", func() {
		k := session.SecretKey([]byte{})
		It("must be 32 bytes long", func() {
			s := k.Sign(k)
			Expect(s).To(BeEmpty())
		})
	})
})

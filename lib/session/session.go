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

package session

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	sessionIdKey = "_ID"
	hashLen      = 32
)

var (
	MalformedSessionDataError        = errors.New("malformed session data")
	SignatureVerificationFailedError = errors.New("signature verification failed")
)

// Session represents a session which will be stored in the session cookies.
type Session map[string]string

// Id returns the session ID. If there isn't one, it generates it.
func (s Session) Id() string {
	if id, ok := s[sessionIdKey]; ok {
		return id
	}

	buf := make([]byte, 32)
	rand.Read(buf)

	s[sessionIdKey] = hex.EncodeToString(buf)

	return s[sessionIdKey]
}

func (s Session) Reset() {
	for k := range s {
		delete(s, k)
	}
}

func EncodeSession(s Session, key SecretKey) string {
	buf := bytes.NewBuffer(nil)
	for k, v := range s {
		if strings.Contains(k, "\x00") {
			panic("a session key cannot contain a 0 byte")
		}
		if strings.Contains(v, "\x00") {
			panic("a session value cannot contain a 0 byte")
		}

		buf.WriteByte(0)
		buf.WriteString(k)
		buf.WriteByte(0)
		buf.WriteString(v)
	}

	data := buf.Bytes()

	encoded := ""

	if len(data) > 1 {
		signature := key.Sign(data[1:])
		encoded = hex.EncodeToString(signature) + hex.EncodeToString(data)
	}

	return encoded
}

func DecodeSession(encoded string, key SecretKey) (Session, error) {
	b, err := hex.DecodeString(encoded)
	if err != nil {
		return make(Session), err
	}

	sess, err := readStringPairs(b, key)
	if err != nil {
		return make(Session), err
	}

	return sess, nil
}

func readStringPairs(b []byte, key SecretKey) (Session, error) {
	pieces, err := readPieces(b, key)
	if err != nil {
		return nil, err
	}
	if len(pieces)%2 == 1 {
		return nil, MalformedSessionDataError
	}

	sess := make(Session)

	for i := 0; i < len(pieces); i += 2 {
		sess[pieces[i]] = pieces[i+1]
	}

	return sess, nil
}

func readPieces(b []byte, key SecretKey) ([]string, error) {
	if len(b) < hashLen+1 {
		return nil, MalformedSessionDataError
	}

	start := hashLen + 1

	if key != nil && !key.Verify(b[start:], b[:start-1]) {
		return nil, SignatureVerificationFailedError
	}

	remaining := b[start:]
	strs := make([]string, countStringPairs(remaining))
	currentString := 0
	for {
		term, end := findNextTerminator(remaining)
		strs[currentString] = string(remaining[:term])
		currentString++

		if end {
			break
		}

		remaining = remaining[term+1:]
	}

	return strs, nil
}

func countStringPairs(slice []byte) int {
	c := 0
	for i := 0; i < len(slice); i++ {
		if slice[i] == 0 {
			c++
		}
	}

	return c + 1
}

func findNextTerminator(slice []byte) (int, bool) {
	for i := 0; i < len(slice); i++ {
		if slice[i] == 0 {
			return i, false
		}
	}

	return len(slice), true
}

// SecretKey will be used to sign and verify the cookies.
type SecretKey []byte

func (s SecretKey) Sign(message []byte) []byte {
	if len(s) != 32 {
		return []byte{}
	}

	mac := hmac.New(sha256.New, s)
	mac.Write(message)
	return mac.Sum(nil)
}

func (s SecretKey) Verify(message []byte, signature []byte) bool {
	return hmac.Equal([]byte(signature), []byte(s.Sign(message)))
}

func MustParse(encoded string) SecretKey {
	b, err := hex.DecodeString(encoded)
	if err != nil {
		panic(err)
	}

	return SecretKey(b)
}

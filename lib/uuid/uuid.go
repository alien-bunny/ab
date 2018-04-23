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

package uuid

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql/driver"

	gouuid "github.com/satori/go.uuid"
)

type UUID [16]byte

var Nil = UUID{}

// Generates a signed UUID.
//
// The key must be 64 bytes long.
func Generate(key []byte) UUID {
	u := UUID(gouuid.NewV4())

	sum := hmacsum(u[:12], key)
	copy(u[12:], sum)

	return u
}

func hmacsum(msg []byte, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return mac.Sum(nil)
}

// Verifies a signed UUID.
//
// The key must be 64 bytes long.
func (u UUID) Verify(key []byte) bool {
	sum := hmacsum(u[:12], key)
	return hmac.Equal(u[12:], sum[:4])
}

// Parses and verifies a signed UUID.
//
// The key must be 64 bytes long.
func ParseAndVerify(key []byte, input string) bool {
	u, uerr := gouuid.FromString(input)
	if uerr != nil {
		return false
	}
	return UUID(u).Verify(key)
}

func (u UUID) IsNil() bool {
	return Equal(u, Nil)
}

func Equal(u1, u2 UUID) bool {
	return gouuid.Equal(gouuid.UUID(u1), gouuid.UUID(u2))
}

func FromBytes(input []byte) (UUID, error) {
	u, err := gouuid.FromBytes(input)
	return UUID(u), err
}

func FromByteOrNil(input []byte) UUID {
	return UUID(gouuid.FromBytesOrNil(input))
}

func FromString(input string) (UUID, error) {
	u, err := gouuid.FromString(input)
	return UUID(u), err
}

func FromStringOrNil(input string) UUID {
	return UUID(gouuid.FromStringOrNil(input))
}

func (u UUID) Bytes() []byte {
	return u[:]
}

func (u UUID) MarshalBinary() ([]byte, error) {
	return gouuid.UUID(u).MarshalBinary()
}

func (u UUID) MarshalText() ([]byte, error) {
	return gouuid.UUID(u).MarshalText()
}

func (u *UUID) Scan(src interface{}) error {
	return (*gouuid.UUID)(u).Scan(src)
}

func (u UUID) String() string {
	return gouuid.UUID(u).String()
}

func (u *UUID) UnmarshalBinary(data []byte) error {
	return (*gouuid.UUID)(u).UnmarshalBinary(data)
}

func (u *UUID) UnmarshalText(text []byte) error {
	return (*gouuid.UUID)(u).UnmarshalText(text)
}

func (u UUID) Value() (driver.Value, error) {
	return gouuid.UUID(u).Value()
}

func (u UUID) Variant() byte {
	return gouuid.UUID(u).Variant()
}

func (u UUID) Version() byte {
	return gouuid.UUID(u).Version()
}

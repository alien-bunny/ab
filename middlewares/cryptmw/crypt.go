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

package cryptmw

import (
	"crypto/cipher"
	"net/http"

	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/logmw"
)

const (
	MiddlewareDependencyCrypt = "*cryptmw.CryptMiddleware"
	cryptKey                  = "abcrypt"
)

func Encrypt(r *http.Request, msg []byte) []byte {
	return util.Encrypt(getCipher(r), msg)
}

func Decrypt(r *http.Request, msg []byte) []byte {
	decrypted, err := util.Decrypt(getCipher(r), msg)
	if err != nil {
		logmw.Warn(r, "crypt middleware", logmw.CategoryInputError).Log("error", err)
		return []byte{}
	}

	return decrypted
}

func EncryptString(r *http.Request, msg string) string {
	return util.EncryptString(getCipher(r), msg)
}

func DecryptString(r *http.Request, msg string) string {
	decrypted, err := util.DecryptString(getCipher(r), msg)
	if err != nil {
		logmw.Warn(r, "crypt middleware", logmw.CategoryInputError).Log("error", err)
		return ""
	}

	return decrypted
}

func getCipher(r *http.Request) cipher.AEAD {
	return r.Context().Value(cryptKey).(*CryptMiddleware).aeadCipher
}

type CryptMiddleware struct {
	aeadCipher cipher.AEAD
}

func (mw *CryptMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = util.SetContext(r, cryptKey, mw)
		next.ServeHTTP(w, r)
	})
}

func (mw *CryptMiddleware) Dependencies() []string {
	return []string{
		logmw.MiddlewareDependencyLog,
	}
}

func NewCryptMiddleware(key []byte) (*CryptMiddleware, error) {
	var err error
	mw := &CryptMiddleware{}
	mw.aeadCipher, err = util.CreateCipher(key)
	if err != nil {
		return nil, err
	}

	return mw, nil
}

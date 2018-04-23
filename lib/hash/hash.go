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

package hash

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/scrypt"
)

var (
	PASSWORD_HASH_SALT_LENGTH = 32
	PASSWORD_HASH_N           = 32768
	PASSWORD_HASH_R           = 8
	PASSWORD_HASH_P           = 1
	PASSWORD_HASH_KEYLEN      = 64

	hashVerifiers = map[string]func(pw, hash string) (bool, error){
		"scrypt": scryptVerify,
	}
)

func DefaultHashPassword(pw string) (string, error) {
	return HashPassword(pw,
		PASSWORD_HASH_SALT_LENGTH,
		PASSWORD_HASH_N,
		PASSWORD_HASH_R,
		PASSWORD_HASH_P,
		PASSWORD_HASH_KEYLEN,
	)
}

func HashPassword(pw string, saltlen, n, r, p, keylen int) (string, error) {
	salt := make([]byte, saltlen)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return "", err
	}

	hash, err := scrypt.Key([]byte(pw), salt, n, r, p, keylen)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("scrypt$%s$%d$%d$%d$%s",
		hex.EncodeToString(salt),
		n, r, p,
		hex.EncodeToString(hash),
	), nil
}

func VerifyPassword(pw, hash string) (bool, error) {
	parts := strings.SplitN(hash, "$", 2)
	if len(parts) != 2 {
		return false, errors.New("invalid hash")
	}

	alg := parts[0]

	if fn, ok := hashVerifiers[alg]; ok {
		return fn(pw, hash)
	} else {
		return false, errors.New("unknown hash algorithm: " + alg)
	}
}

func scryptVerify(pw, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false, err
	}

	n, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, err
	}

	r, err := strconv.Atoi(parts[3])
	if err != nil {
		return false, err
	}

	p, err := strconv.Atoi(parts[4])
	if err != nil {
		return false, err
	}

	pwhash, err := hex.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	newhash, err := scrypt.Key([]byte(pw), salt, n, r, p, len(pwhash))
	if err != nil {
		return false, err
	}

	if len(pwhash) != len(newhash) {
		return false, nil
	}

	ok := true
	for i := 0; i < len(pwhash); i++ {
		ok = ok && (pwhash[i] == newhash[i])
	}

	return ok, nil
}

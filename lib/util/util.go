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

package util

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/alien-bunny/ab/lib/uuid"
	"golang.org/x/crypto/ssh"
)

func init() {
	seedRandom()
}

// seedRandom sets a random seed for math/rand. This way functions that use the math random generator gets more random.
func seedRandom() {
	b := make([]byte, 8)
	rand.Read(b)
	s := binary.BigEndian.Uint64(b)
	mrand.Seed(int64(s))
}

// GeneratePlaceholders generates placeholders from start to end for an SQL query.
func GeneratePlaceholders(start, end uint) string {
	ret := ""
	if start == end {
		return ret
	}
	for i := start; i < end; i++ {
		ret += ", $" + strconv.Itoa(int(i))
	}

	return ret[2:]
}

// StringSliceToInterfaceSlice converts a string slice into an interface{} slice.
func StringSliceToInterfaceSlice(s []string) []interface{} {
	is := make([]interface{}, len(s))
	for i, d := range s {
		is[i] = d
	}

	return is
}

func UUIDSliceToInterface(s []uuid.UUID) []interface{} {
	is := make([]interface{}, len(s))
	for i, d := range s {
		is[i] = d
	}

	return is
}

// ResponseBodyToString reads the whole response body and converts it to a string.
func ResponseBodyToString(r *http.Response) string {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(b)
}

const keySize = 2048

// GenerateKey generates an RSA key (2048 bit) and encodes it using PEM.
func GenerateKey() string {
	prikey, _ := rsa.GenerateKey(rand.Reader, keySize)
	return string(MarshalPrivateKey(prikey))
}

// MarshalPrivateKey converts a private key to a PEM encoded string.
func MarshalPrivateKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(key),
	})
}

// UnmarshalPrivateKey converts a PEM encoded string into a private key.
//
// Returns nil on failure.
func UnmarshalPrivateKey(key []byte) *rsa.PrivateKey {
	marshaled, _ := pem.Decode(key)
	prikey, err := x509.ParsePKCS1PrivateKey(marshaled.Bytes)
	if err != nil {
		return nil
	}

	return prikey
}

// GetPublicKey gets the public part of a private key in OpenSSL format.
func GetPublicKey(key *rsa.PrivateKey) string {
	pkey, _ := ssh.NewPublicKey(&key.PublicKey)
	marshalled := pkey.Marshal()

	return "ssh-rsa " + base64.StdEncoding.EncodeToString(marshalled) + "\n"
}

func CreateCipher(key []byte) (cipher.AEAD, error) {
	aescipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(aescipher)
}

func Encrypt(aeadCipher cipher.AEAD, msg []byte) []byte {
	nonce := make([]byte, aeadCipher.NonceSize())
	io.ReadFull(rand.Reader, nonce)

	buf := bytes.NewBuffer(nil)
	buf.Write(nonce)

	encrypted := aeadCipher.Seal(nil, nonce, msg, nil)
	buf.Write(encrypted)

	return buf.Bytes()
}

func Decrypt(aeadCipher cipher.AEAD, msg []byte) ([]byte, error) {
	noncelen := aeadCipher.NonceSize()
	nonce := msg[:noncelen]
	encrypted := msg[noncelen:]

	return aeadCipher.Open(nil, nonce, encrypted, nil)
}

func EncryptString(aeadCipher cipher.AEAD, msg string) string {
	if msg == "" {
		return msg
	}

	return base64.StdEncoding.EncodeToString(Encrypt(aeadCipher, []byte(msg)))
}

func DecryptString(aeadCipher cipher.AEAD, msg string) (string, error) {
	if msg == "" {
		return "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return "", err
	}

	decrypted, err := Decrypt(aeadCipher, decoded)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

var colorCodeRegex = regexp.MustCompile(`\[[0-9;]+m`)

func StripTerminalColorCodes(s string) string {
	return colorCodeRegex.ReplaceAllString(s, "")
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandomString generates a random string of a given length.
func RandomString(length int) string {
	b := make([]byte, length)

	for i, cache, remain := length-1, mrand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = mrand.Int63(), letterIdxMax
		}

		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}

		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// RandomSecret generates a random secret of length bytes long.
//
// The returned data will be hex encoded, so it will be length*2
// characters.
func RandomSecret(length int) string {
	buf := make([]byte, length)
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

// SetContext sets a value in the context of *http.Request, and returns a new one with the updated context.
func SetContext(r *http.Request, key, value interface{}) *http.Request {
	ctx := context.WithValue(r.Context(), key, value)
	return r.WithContext(ctx)
}

// RedirectDestination constucts an URL to the redirect destination.
//
// The redirect destination is read from the destination URL parameter.
func RedirectDestination(r *http.Request) string {
	return "/" + r.URL.Query().Get("destination")
}

func TestServerAddress() string {
	return fmt.Sprintf("localhost:%d", 30000+mrand.Intn(10000))
}

// BuildLink buils a link from the current request.
func BuildLink(r *http.Request, path string, query map[string]string) *url.URL {
	u := &url.URL{}
	u.Scheme = r.URL.Scheme
	u.Host = r.URL.Host
	if path == "" {
		u.Path = r.URL.Path
	} else {
		u.Path = path
	}
	for k, v := range query {
		u.Query().Set(k, v)
	}

	return u
}

var _ http.Hijacker = ResponseWriterWrapper{}
var _ http.Flusher = ResponseWriterWrapper{}
var _ http.Pusher = ResponseWriterWrapper{}

type ResponseWriterWrapper struct {
	http.ResponseWriter
}

func (w ResponseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}

	return nil, nil, http.ErrNotSupported
}

func (w ResponseWriterWrapper) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w ResponseWriterWrapper) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}

	return http.ErrNotSupported
}

// GenerateCertificate generates a cerificate for a host.
//
// The first return value is the ceritificate, the second is the key. Both are PEM encoded.
func GenerateCertificate(host, organization string) (string, string) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365 * 10)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{host},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		panic(err)
	}

	certOut := bytes.NewBuffer(nil)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer(nil)
	pem.Encode(keyOut, pemBlockForKey(priv))

	return string(certOut.Bytes()), string(keyOut.Bytes())
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}

}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			panic(err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	}

	return nil
}

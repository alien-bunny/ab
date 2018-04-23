package cryptmw_test

import (
	"crypto/rand"
	"net/http"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/cryptmw"
	"github.com/alien-bunny/ab/middlewares/logmw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Crypt middleware", func() {
	buf := make([]byte, 32)
	rand.Read(buf)

	logger := abtest.GetLogger()
	cmw, err := cryptmw.NewCryptMiddleware(buf)

	if err != nil {
		panic(err)
	}

	stack := middleware.NewStack(nil)
	stack.Push(logmw.New(logger))
	stack.Push(cmw)

	It("should encrypt and decrypt a string", func() {
		msg := util.RandomSecret(1024)
		encrypted := ""

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			encrypted = cryptmw.EncryptString(r, msg)
			Expect(encrypted).NotTo(BeEmpty())
		})
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			decrypted := cryptmw.DecryptString(r, encrypted)
			Expect(decrypted).NotTo(BeEmpty())
			Expect(decrypted).To(Equal(msg))
		})
	})

	It("should encrypt and decrypt a data", func() {
		msg := make([]byte, 4096)
		rand.Read(msg)
		var encrypted []byte

		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			encrypted = cryptmw.Encrypt(r, msg)
			Expect(encrypted).NotTo(BeEmpty())
		})
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			decrypted := cryptmw.Decrypt(r, encrypted)
			Expect(decrypted).NotTo(BeEmpty())
			Expect(decrypted).To(Equal(msg))
		})
	})
})

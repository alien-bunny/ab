package cryptmw_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCryptmw(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cryptmw Suite")
}

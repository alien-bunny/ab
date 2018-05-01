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

package collectionloader_test

import (
	"io/ioutil"
	"reflect"

	"github.com/alien-bunny/ab/lib/collectionloader"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collectionloader", func() {
	logger := log.NewDevLogger(ioutil.Discard)
	conf := config.NewStore(logger)
	cl := collectionloader.NewDirectory(".", map[string]string{
		"test": "fixtures",
	}, true)
	conf.RegisterSchema("test", reflect.TypeOf(test{}))
	conf.AddCollectionLoaders(cl)

	It("should find the configuration file", func() {
		testInterface, err := conf.Get("test").Get("test")
		Expect(err).NotTo(HaveOccurred())
		Expect(testInterface).NotTo(BeNil())
		testData := testInterface.(test)
		Expect(testData.A).To(Equal(5))
		Expect(testData.B).To(Equal("asdf"))
	})

	It("should return an error when the directory does not exists", func() {
		testInterface := conf.Get("asdf")
		Expect(testInterface).To(BeNil())
	})

	It("should return an error when it is not a directory", func() {
		testInterface := conf.Get("collectionloader_suite_test.go")
		Expect(testInterface).To(BeNil())
	})
})

type test struct {
	A int
	B string
}

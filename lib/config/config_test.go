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

package config_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type test struct {
	A int
	B string
	C bool
	D struct {
		E int
		F float64
	}
	G string
}

var _ = Describe("Config", func() {
	expected := testExample()

	c := config.NewStore(log.NewDevLogger(ioutil.Discard))
	c.RegisterSchema("test.*", reflect.TypeOf(test{}))
	var entries []TableEntry
	for _, t := range []string{"test.0", "test.1", "test.2", "test.3"} {
		entries = append(entries, Entry(t, t))
		os.Setenv("CONFIG_"+strings.ToUpper(t)+"_G", "zxcvbn")
	}

	defaultCollection := config.NewCollection()
	ep := config.NewEnvConfigProvider()
	ep.Prefix = "CONFIG"
	ep.Reset()
	dp := config.NewDirectoryConfigProvider("fixtures/config", true)
	registerFileTypes(dp)
	defaultCollection.AddProviders(ep, dp)
	c.AddCollection("config", defaultCollection)

	DescribeTable("load different types of config",
		func(key string) {
			v, err := c.Get("config").Get(key)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal(expected))
		},
		entries...,
	)

	It("collection not found is handled", func() {
		Expect(c.Get("test")).To(BeNil())
	})
})

var _ = Describe("Writable config", func() {
	entries := []TableEntry{
		Entry("JSON", &config.JSON{}),
		Entry("YAML", &config.YAML{}),
		Entry("TOML", &config.TOML{}),
		Entry("XML", &config.XML{}),
	}

	DescribeTable("save different types of config",
		func(ft config.FileType) {
			ep := config.NewEnvConfigProvider()
			ep.Prefix = ""
			ep.Reset()

			tmpdir, err := ioutil.TempDir("", "abtest")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tmpdir)

			dp := config.NewDirectoryConfigProvider("fixtures/config", true)
			registerFileTypes(dp)
			dpw := config.NewDirectoryConfigProvider(tmpdir, false)
			dpw.RegisterFiletype(ft)
			registerFileTypes(dpw)

			collection := config.NewCollection()
			collection.AddProviders(ep, dpw, dp)

			c := config.NewStore(log.NewDevLogger(ioutil.Discard))
			c.RegisterSchema("test.*", reflect.TypeOf(test{}))
			c.AddCollection("config", collection)

			t := saveValue(c, "qwer")
			checkValue(tmpdir, ft, t)
			collection.ClearCache()
		},
		entries...,
	)
})

var _ = Describe("Only readonly providers", func() {
	c := config.NewStore(log.NewDevLogger(ioutil.Discard))
	c.RegisterSchema("test.*", reflect.TypeOf(test{}))

	collection := config.NewCollection()
	dp := config.NewDirectoryConfigProvider("fixtures/config", true)
	registerFileTypes(dp)
	collection.AddProviders(dp)

	c.AddCollection("config", collection)

	It("should error when no providers can save the config", func() {
		ti, saver, err := c.GetWritable("config").GetWritable("test.0")
		t := ti.(test)
		Expect(err).NotTo(HaveOccurred())

		err = saver.Save(t)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Collection loaders", func() {
	c := config.NewStore(log.NewDevLogger(ioutil.Discard))
	c.RegisterSchema("test", reflect.TypeOf(test{}))

	mp := config.NewMemoryConfigProvider()
	collection := config.NewCollection()
	collection.SetTemporary(true)
	collection.AddProviders(mp)
	var loader config.CollectionLoaderFunc = func(name string) (*config.Collection, error) {
		if name == "test" {
			return collection, nil
		}
		return nil, config.CollectionNotFoundError{Name: name}
	}
	c.AddCollectionLoaders(loader)

	BeforeEach(func() {
		collection.ClearCache()
		mp.Reset()
		c.RemoveTemporary()
	})

	It("should load a collection and find a value", func() {
		_, saver, err := c.GetWritable("test").GetWritable("test")
		Expect(err).NotTo(HaveOccurred())
		err = saver.Save(testExample())
		Expect(err).NotTo(HaveOccurred())

		collection.ClearCache()

		res, err := c.Get("test").Get("test")
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(testExample()))
	})

	It("should not find a value that doesn't exists", func() {
		res, err := c.Get("test").Get("test")
		Expect(res).To(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("Schema can't be added twice", func() {
	c := config.NewStore(log.NewDevLogger(ioutil.Discard))
	c.RegisterSchema("test", reflect.TypeOf(test{}))

	It("should panic when a schema is registered twice", func() {
		Expect(func() {
			c.RegisterSchema("test", reflect.TypeOf(test{}))
		}).NotTo(Panic())
		Expect(func() {
			c.RegisterSchema("test", reflect.TypeOf(struct{}{}))
		}).To(Panic())
	})
})

func testExample() test {
	example := test{
		A: 5,
		B: "asdf",
		C: true,
		G: "zxcvbn",
	}
	example.D.E = -2
	example.D.F = -1.2

	return example
}

func checkValue(tmpdir string, ft config.FileType, t test) {
	fn := filepath.FromSlash(path.Join(tmpdir, "test.0")) + "." + ft.Extensions()[0]
	f, err := os.Open(fn)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	ut := test{}
	ft.Unmarshal(f, &ut)
	Expect(ut).To(Equal(t))
}

func saveValue(c *config.Store, value string) test {
	v, saver, err := c.GetWritable("config").GetWritable("test.0")
	Expect(err).NotTo(HaveOccurred())
	t := v.(test)
	t.G = value
	err = saver.Save(t)
	Expect(err).NotTo(HaveOccurred())
	return t
}

func registerFileTypes(dp *config.DirectoryConfigProvider) {
	dp.RegisterFiletype(&config.YAML{})
	dp.RegisterFiletype(&config.JSON{})
	dp.RegisterFiletype(&config.TOML{})
	dp.RegisterFiletype(&config.XML{})
}

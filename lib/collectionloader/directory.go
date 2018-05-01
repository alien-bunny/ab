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

package collectionloader

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/alien-bunny/ab/lib/config"
)

type Directory struct {
	base     string
	conf     map[string]string
	readOnly bool
}

func NewDirectory(base string, conf map[string]string, readOnly bool) *Directory {
	return &Directory{
		base:     base,
		conf:     conf,
		readOnly: readOnly,
	}
}

func (d *Directory) Load(name string) (*config.Collection, error) {
	if alias, found := d.conf[name]; found {
		name = alias
	}

	dir := filepath.Join(d.base, name)

	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, config.CollectionNotFoundError{Name: dir}
	}

	c := config.NewCollection()
	c.SetTemporary(true)

	e := config.NewEnvConfigProvider()
	e.Prefix = "SITE_" + strings.ToUpper(name)

	p := config.NewDirectoryConfigProvider(dir, d.readOnly)
	p.RegisterFiletype(&config.JSON{})
	p.RegisterFiletype(&config.YAML{})
	p.RegisterFiletype(&config.TOML{})
	p.RegisterFiletype(&config.XML{})

	c.AddProviders(e, p)

	return c, nil
}

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

package config

import (
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/alien-bunny/ab/lib/errors"
)

var _ WritableProvider = &DirectoryConfigProvider{}

type FileType interface {
	Extensions() []string
	Unmarshal(stream io.Reader, v interface{}) error
	Marshal(stream io.Writer, v interface{}) error
}

type DirectoryConfigProvider struct {
	base      string
	readOnly  bool
	fileTypes []FileType
}

func NewDirectoryConfigProvider(base string, readOnly bool) *DirectoryConfigProvider {
	return &DirectoryConfigProvider{
		base:     base,
		readOnly: readOnly,
	}
}

func (d *DirectoryConfigProvider) RegisterFiletype(t FileType) {
	d.fileTypes = append(d.fileTypes, t)
}

func (d *DirectoryConfigProvider) basenameForKey(key string) string {
	return filepath.FromSlash(path.Join(d.base, key))
}

func (d *DirectoryConfigProvider) exists(key string) (FileType, string) {
	name := d.basenameForKey(key)
	for _, t := range d.fileTypes {
		for _, ext := range t.Extensions() {
			fn := name + "." + ext
			if _, err := os.Stat(fn); err == nil {
				return t, fn
			}
		}
	}

	return nil, ""
}

func (d *DirectoryConfigProvider) Has(key string) bool {
	_, fn := d.exists(key)
	return fn != ""
}

func (d *DirectoryConfigProvider) Unmarshal(key string, v interface{}) error {
	ft, fn := d.exists(key)
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	return ft.Unmarshal(f, v)
}

func (d *DirectoryConfigProvider) CanSave(key string) bool {
	return !d.readOnly
}

func (d *DirectoryConfigProvider) Save(key string, v interface{}) error {
	var f *os.File
	var err error

	ft, fn := d.exists(key)
	if fn == "" { // file does not exists
		if len(d.fileTypes) == 0 {
			return errors.New("no configured file type for this directory config provider")
		}
		name := d.basenameForKey(key) + "." + d.fileTypes[0].Extensions()[0]
		ft = d.fileTypes[0]
		f, err = os.Create(name)
	} else { // file exists
		f, err = os.Open(fn)
	}
	if err != nil {
		return err
	}
	defer f.Close()

	return ft.Marshal(f, v)
}

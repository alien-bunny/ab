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

package matcher

import (
	"strings"
)

type Matcher struct {
	separator string
	tree      *item
}

func NewMatcher(separator string) *Matcher {
	return &Matcher{
		separator: separator,
		tree:      newItem(),
	}
}

func (m *Matcher) get(path string, create bool) *item {
	parts := strings.Split(path, m.separator)
	return m.tree.get(parts, create)
}

func (m *Matcher) Get(path string) interface{} {
	item := m.get(path, false)
	if item == nil {
		return nil
	}

	return item.content
}

func (m *Matcher) Set(path string, content interface{}) {
	item := m.get(path, true)
	item.content = content
}

type item struct {
	children map[string]*item
	wildcard *item
	content  interface{}
}

func newItem() *item {
	return &item{
		children: make(map[string]*item),
		wildcard: nil,
	}
}

func (i *item) get(path []string, create bool) *item {
	if len(path) == 0 {
		return i
	}

	current := path[0]
	if current == "*" {
		if i.wildcard != nil {
			return i.wildcard.get(path[1:], create)
		}
		if create {
			i.wildcard = newItem()
			return i.wildcard.get(path[1:], create)
		}
	}

	if childItem, found := i.children[current]; found {
		return childItem.get(path[1:], create)
	}

	if create {
		i.children[current] = newItem()
		return i.children[current].get(path[1:], create)
	}

	if i.wildcard != nil {
		return i.wildcard.get(path[1:], create)
	}

	return nil
}

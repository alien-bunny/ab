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

package errors_test

import (
	stderrors "errors"
	"net/http"

	"github.com/alien-bunny/ab/lib/errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error handler library", func() {
	const (
		errmsg     = "qwerty"
		verbosemsg = "asdfghzxcvbn"
	)

	Describe("The Fail function", func() {
		It("must trigger a panic", func() {
			Expect(func() {
				errors.Fail(http.StatusInternalServerError, errors.New(""))
			}).To(Panic())
		})
	})

	Describe("The wrapped error", func() {
		wrappedErr := errors.NewError(errmsg, verbosemsg, nil)
		It("should wrap an error message with the verbose message", func() {
			Expect(wrappedErr.Error()).To(Equal(errmsg))
			Expect(wrappedErr.UserError(nil)).To(Equal(verbosemsg))
		})

		wrappedVerboseOnlyErr := errors.NewError("", verbosemsg, nil)
		It("should wrap an empty error message, replacing it with the verbose error", func() {
			Expect(wrappedVerboseOnlyErr.Error()).To(Equal(verbosemsg))
			Expect(wrappedVerboseOnlyErr.UserError(nil)).To(Equal(verbosemsg))
		})
	})

	Describe("The panic type", func() {
		p := errors.Panic{
			Err: stderrors.New(errmsg),
		}
		pv := errors.Panic{
			Err: errors.NewError(errmsg, verbosemsg, nil),
		}
		It("should wrap the error message", func() {
			Expect(p.Error()).To(Equal(errmsg))
			Expect(p.String()).To(Equal(errmsg))
			Expect(p.UserError(nil)).To(BeZero())
			Expect(pv.Error()).To(Equal(errmsg))
			Expect(pv.UserError(nil)).To(Equal(verbosemsg))
		})
	})
})

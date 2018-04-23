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

package errors

import (
	"errors"
)

// Error extends the built-in error interface with a message that is displayed to the end user.
type Error interface {
	// Error that is displayed in the logs and debug messages. Should contain diagnostical information.
	Error() string
	// Error that is displayed to the end user.
	UserError(t func(string, map[string]string) string) string
}

var _ Error = &errorWrapper{}

type errorWrapper struct {
	error
	message string
	params  map[string]string
}

func (ew *errorWrapper) UserError(t func(string, map[string]string) string) string {
	if t != nil {
		return t(ew.message, ew.params)
	}
	return ew.message
}

func (ew *errorWrapper) Cause() error {
	return ew.error
}

// Wrap wraps an error message into a Error.
func Wrap(err error, message string, params map[string]string) Error {
	return &errorWrapper{
		error:   err,
		message: message,
		params:  params,
	}
}

// NewError creates a new verbose error message.
//
// If err is an empty string, then message will be used it instead.
func NewError(err, message string, params map[string]string) Error {
	if err == "" {
		err = message
	}

	return Wrap(errors.New(err), message, params)
}

// New is a replacement function for errors.New().
//
// This function constructs a Error where both the diagnostic error and the end user error is the same.
func New(message string) error {
	return NewError(message, message, nil)
}

var _ Error = Panic{}

// Calls HandleError on the Error object inside the request context.
func Fail(code int, err error) {
	panic(Panic{
		Code: code,
		Err:  err,
	})
}

// Panic is a custom panic data structure for the ErrorHandler.
type Panic struct {
	Code          int
	Err           error
	StackTrace    string
	DisplayErrors bool
}

func (p Panic) Error() string {
	return p.Err.Error()
}

func (p Panic) String() string {
	return p.Err.Error()
}

func (p Panic) UserError(t func(string, map[string]string) string) string {
	if ve, ok := p.Err.(Error); ok {
		return ve.UserError(t)
	}

	return ""
}

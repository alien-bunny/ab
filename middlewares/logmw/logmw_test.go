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

package logmw_test

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/alien-bunny/ab/lib/abtest"
	"github.com/alien-bunny/ab/lib/log"
	"github.com/alien-bunny/ab/lib/middleware"
	"github.com/alien-bunny/ab/lib/util"
	"github.com/alien-bunny/ab/middlewares/logmw"
	"github.com/alien-bunny/ab/middlewares/requestmw"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Log Middleware", func() {
	out := bytes.NewBuffer(nil)

	logger := log.NewJSONLogger(out, level.AllowAll())

	stack := middleware.NewStack(nil)
	stack.Push(requestmw.NewRequestIDMiddleware())
	stack.Push(logmw.New(logger))

	It("should record log messages in the request buffer", func() {
		msg := util.RandomString(64)
		abtest.TestMiddleware(stack, func(w http.ResponseWriter, r *http.Request) {
			m := map[string]func(*http.Request, interface{}, interface{}) kitlog.Logger{
				"debug": logmw.Debug,
				"info":  logmw.Info,
				"warn":  logmw.Warn,
				"error": logmw.Error,
			}

			for lvl, logger := range m {
				logger(r, "logmw", "test").Log("msg", msg)
				assertLog(lvl, msg, out)
			}
		})
	})
})

func assertLog(lvl string, msg string, buf *bytes.Buffer) {
	loggedBytes := buf.Bytes()
	buf.Reset()
	data := make(map[string]string)
	Expect(json.Unmarshal(loggedBytes, &data)).NotTo(HaveOccurred())
	Expect(data["level"]).To(Equal(lvl))
	Expect(data["category"]).To(Equal("test"))
	Expect(data["component"]).To(Equal("logmw"))
	Expect(data["msg"]).To(Equal(msg))
}

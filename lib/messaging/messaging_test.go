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

package messaging_test

import (
	"net/smtp"
	"text/template"

	"github.com/alien-bunny/ab/lib/messaging"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type data struct {
	Name string
}

type target string

func (t target) Address() string {
	return string(t)
}

var _ = Describe("Messaging", func() {
	body := ""
	sender := messaging.NewSMTPMessageSender("from@example.com", "smtp.example.com", nil)
	sender.SendMail = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		body = string(msg)
		return nil
	}
	sender.AddTemplate(template.Must(template.New("test").Parse(`Hello, {{.Name}}!`)))

	It("should not find templates that does not exists", func() {
		err := sender.Send("", nil, nil)
		Expect(err).To(HaveOccurred())
	})

	It("should properly render the message and send it through SMTP", func() {
		err := sender.Send("test", target("test@example.com"), data{
			Name: "World",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(Equal("Hello, World!"))
	})
})

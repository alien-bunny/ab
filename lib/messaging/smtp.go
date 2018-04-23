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

package messaging

import (
	"bytes"
	"net/smtp"
	"text/template"

	"github.com/alien-bunny/ab/lib/errors"
)

var _ MessageSender = &SMTPMessageSender{}

type SMTPMessageSender struct {
	from       string
	serverAddr string
	smtpAuth   smtp.Auth
	templates  map[string]*template.Template
	SendMail   func(string, smtp.Auth, string, []string, []byte) error
}

func NewSMTPMessageSender(from, serverAddr string, smtpAuth smtp.Auth) *SMTPMessageSender {
	return &SMTPMessageSender{
		from:       from,
		serverAddr: serverAddr,
		smtpAuth:   smtpAuth,
		templates:  make(map[string]*template.Template),
		SendMail:   smtp.SendMail,
	}
}

func (s *SMTPMessageSender) AddTemplate(tpl *template.Template) *SMTPMessageSender {
	s.templates[tpl.Name()] = tpl
	return s
}

func (s *SMTPMessageSender) Send(msgtype string, target MessageTarget, data interface{}) error {
	tpl, ok := s.templates[msgtype]
	if !ok {
		return errors.New("template not found")
	}

	buf := bytes.NewBuffer(nil)
	if err := tpl.Execute(buf, data); err != nil {
		return err
	}

	return s.SendMail(s.serverAddr, s.smtpAuth, s.from, []string{target.Address()}, buf.Bytes())
}

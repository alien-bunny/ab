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

package ab_test

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alien-bunny/ab"
	"github.com/alien-bunny/ab/lib/config"
	"github.com/alien-bunny/ab/lib/server"
	"github.com/alien-bunny/ab/lib/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hop should start without error and serve the index page", func() {
	addr := setHostAndPort()
	It("should be able to start a server", func() {
		errch := ab.Hop(func(conf *config.Store, s *server.Server) error {
			return nil
		}, nil, "./fixtures")
		select {
		case err := <-errch:
			Expect(err).NotTo(HaveOccurred())
		case <-time.After(time.Second / 10):
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		indexfile, err := ioutil.ReadFile("./fixtures/index.html")
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("GET", "https://"+addr+"/", nil)
		Expect(err).NotTo(HaveOccurred())

		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		respdata, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(respdata)).To(Equal(string(indexfile)))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		close(errch)
		<-time.After(time.Second / 2)

		_, err = http.Get("https://" + addr)
		Expect(err).To(HaveOccurred())
	})
})

func setHostAndPort() string {
	addr := util.TestServerAddress()
	parts := strings.Split(addr, ":")
	os.Setenv("CONFIG_HOST", parts[0])
	os.Setenv("CONFIG_PORT", parts[1])

	return addr
}

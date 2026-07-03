package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("checkEndpoint", func() {
	var (
		server *httptest.Server
		status int
		body   string
	)

	// startServer spins up a test server that replies with the current
	// status/body. Called from each spec after those are set.
	startServer := func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			fmt.Fprint(w, body)
		}))
	}

	AfterEach(func() {
		if server != nil {
			server.Close()
			server = nil
		}
	})

	DescribeTable("verifying the required field",
		func(respStatus int, respBody, field string, expectValue any, expectErr bool) {
			status, body = respStatus, respBody
			startServer()

			value, _, err := checkEndpoint(server.Client(), server.URL, field)

			if expectErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(expectValue))
		},
		Entry("field present and false", 200, `{"completed": false}`, "completed", false, false),
		Entry("field present and true", 200, `{"completed": true}`, "completed", true, false),
		Entry("numeric field", 200, `{"count": 3}`, "count", float64(3), false),
		Entry("field missing", 200, `{"title":"x"}`, "completed", nil, true),
		Entry("non-200 status", 500, `{}`, "completed", nil, true),
		Entry("invalid json", 200, `not json`, "completed", nil, true),
	)

	It("errors when the endpoint is unreachable", func() {
		status, body = 200, "{}"
		startServer()
		url := server.URL
		server.Close()
		server = nil

		client := &http.Client{Timeout: time.Second}
		_, _, err := checkEndpoint(client, url, "completed")
		Expect(err).To(HaveOccurred())
	})
})

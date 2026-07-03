package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy", func() {
	var (
		server         *httptest.Server
		status         int
		respBody       string
		gotAuth        string
		gotPath        string
		gotBody        string
		gotContentType string
	)

	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotContentType = r.Header.Get("Content-Type")
			gotPath = r.URL.Path
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			w.WriteHeader(status)
			fmt.Fprint(w, respBody)
		}))
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
			server = nil
		}
	})

	Context("when the deploy succeeds", func() {
		It("returns the deploy id from a structured response", func() {
			status, respBody = 200, `{"status":"succeeded","deployId":"abc123","message":"done"}`

			msg, err := deploy(server.Client(), server.URL, "tok-123", "1.2.3")

			Expect(err).NotTo(HaveOccurred())
			Expect(msg).To(ContainSubstring("abc123"))
		})

		It("treats any 2xx with a non-JSON body as success", func() {
			status, respBody = 201, `plain ok`

			msg, err := deploy(server.Client(), server.URL, "tok-123", "1.2.3")

			Expect(err).NotTo(HaveOccurred())
			Expect(msg).To(ContainSubstring("HTTP 201"))
		})

		It("sends a well-formed, authenticated request", func() {
			status, respBody = 200, `{"status":"succeeded"}`

			_, err := deploy(server.Client(), server.URL, "tok-123", "1.2.3")
			Expect(err).NotTo(HaveOccurred())

			Expect(gotPath).To(Equal("/deploy"))
			Expect(gotAuth).To(Equal("Bearer tok-123"))
			Expect(gotContentType).To(Equal("application/json"))

			var req DeployRequest
			Expect(json.Unmarshal([]byte(gotBody), &req)).To(Succeed())
			Expect(req.Version).To(Equal("1.2.3"))
			Expect(req.Service).To(Equal(serviceName))
		})
	})

	Context("when the deploy fails", func() {
		It("errors on a >=400 status", func() {
			status, respBody = 500, `boom`

			_, err := deploy(server.Client(), server.URL, "tok", "1.0.0")
			Expect(err).To(MatchError(ContainSubstring("HTTP 500")))
		})

		It("errors when the body reports status failed", func() {
			status, respBody = 200, `{"status":"failed","message":"bad config"}`

			_, err := deploy(server.Client(), server.URL, "tok", "1.0.0")
			Expect(err).To(MatchError(ContainSubstring("bad config")))
		})

		It("errors when the endpoint is unreachable", func() {
			status, respBody = 200, `{}`
			url := server.URL
			server.Close()
			server = nil

			_, err := deploy(&http.Client{}, url, "tok", "1.0.0")
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("env helpers", func() {
	Describe("getEnv", func() {
		It("returns the value when the variable is set", func() {
			GinkgoT().Setenv("DEPLOY_TEST_VAR", "bar")
			Expect(getEnv("DEPLOY_TEST_VAR", "fallback")).To(Equal("bar"))
		})

		It("returns the fallback when the variable is unset", func() {
			Expect(getEnv("DEPLOY_UNSET_VAR_XYZ", "fallback")).To(Equal("fallback"))
		})
	})

	Describe("requireEnv", func() {
		It("returns the value when the variable is present", func() {
			GinkgoT().Setenv("DEPLOY_REQ_VAR", "value")
			Expect(requireEnv("DEPLOY_REQ_VAR")).To(Equal("value"))
			// The missing-variable branch calls os.Exit(1) and is not asserted here.
		})
	})
})

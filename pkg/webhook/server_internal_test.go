package webhook

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Internal", func() {
	Describe("healthzHandler", func() {
		It("should return 200 OK", func() {
			req, err := http.NewRequest("GET", "/healthz", nil)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(healthzHandler)

			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(Equal("OK"))
		})
	})
})

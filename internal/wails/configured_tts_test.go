package wails

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTTSConfigurationFixture(t *testing.T, model string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			fmt.Fprintf(w, `{"data":[{"id":%q}]}`, model)
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(server.Close)
	return server
}

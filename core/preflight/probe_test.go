package preflight

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbe_OK_2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if err := Probe(srv.URL); err != nil {
		t.Errorf("Probe(2xx) should succeed: %v", err)
	}
}

func TestProbe_AcceptsAny2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // 204
	}))
	defer srv.Close()
	if err := Probe(srv.URL); err != nil {
		t.Errorf("204 should be accepted: %v", err)
	}
}

func TestProbe_NonOKReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := Probe(srv.URL); err == nil {
		t.Error("Probe(5xx) should error")
	}
}

func TestProbe_EmptyURLErrors(t *testing.T) {
	if err := Probe(""); err == nil {
		t.Error("empty URL should error")
	}
	if err := Probe("   "); err == nil {
		t.Error("whitespace URL should error")
	}
}

func TestProbe_TrailingSlashTolerated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if err := Probe(srv.URL + "/"); err != nil {
		t.Errorf("trailing slash should be trimmed: %v", err)
	}
}

func TestProbe_UnreachableReturnsError(t *testing.T) {
	// Port 1 reliably refuses on most systems within the 3s window.
	if err := Probe("http://127.0.0.1:1"); err == nil {
		t.Error("unreachable endpoint should error")
	}
}

package requestip

import (
	"net/http"
	"testing"
)

func TestFromRequestWithForwarded(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://localhost", nil)
	r.RemoteAddr = "10.0.0.2:54433"
	r.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2")
	got := FromRequest(r)
	if got != "1.1.1.1" {
		t.Fatalf("expected 1.1.1.1, got %s", got)
	}
}

func TestFromRequestFallbackRemoteAddr(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://localhost", nil)
	r.RemoteAddr = "10.0.0.2:54433"
	got := FromRequest(r)
	if got != "10.0.0.2" {
		t.Fatalf("expected 10.0.0.2, got %s", got)
	}
}

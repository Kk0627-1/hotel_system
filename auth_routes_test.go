package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtectedRoutesRequireToken(t *testing.T) {
	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	paths := []string{
		"/hotel/create",
		"/hotel/update",
		"/hotel/delete",
		"/hotel/list",
		"/room/create",
		"/room/update",
		"/room/delete",
		"/room/list",
		"/payment/cancel",
		"/payment/list",
		"/payment/bill",
		"/wallet/recharge",
	}
	for _, path := range paths {
		response, err := http.Post(server.URL+path, "application/json", strings.NewReader(`{}`))
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("POST %s status = %d, want %d", path, response.StatusCode, http.StatusUnauthorized)
		}
	}
}

func TestRegisterDoesNotRequireToken(t *testing.T) {
	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	response, err := http.Post(server.URL+"/register", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		t.Fatal("register rejected a request that has no token")
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

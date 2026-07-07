package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatus(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	var body statusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Status != "running" {
		t.Errorf("status = %q, want %q", body.Status, "running")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing permissive CORS header")
	}
}

func TestPreflight(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodOptions, "/open", nil)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Header().Get("Access-Control-Allow-Private-Network") != "true" {
		t.Errorf("missing Access-Control-Allow-Private-Network header")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing Access-Control-Allow-Origin header")
	}
}

func TestOpenErrors(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "nope")

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"invalid json", "{", http.StatusBadRequest, "bad_request"},
		{"empty path", `{"path":""}`, http.StatusBadRequest, "bad_request"},
		{"relative path", `{"path":"foo/bar"}`, http.StatusBadRequest, "bad_request"},
		{"missing path", `{"path":` + mustJSON(missing) + `}`, http.StatusNotFound, "not_found"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/open", strings.NewReader(test.body))
			rec := httptest.NewRecorder()
			newHandler().ServeHTTP(rec, req)

			if rec.Code != test.wantStatus {
				t.Fatalf("status code = %d, want %d (body %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
			var body errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if body.Code != test.wantCode {
				t.Errorf("code = %q, want %q", body.Code, test.wantCode)
			}
			if body.Error == "" {
				t.Errorf("error message is empty")
			}
		})
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

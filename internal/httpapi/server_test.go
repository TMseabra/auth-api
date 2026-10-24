package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	New(nil, nil).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestDecodeJSONRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	for _, body := range []string{
		`{"email":"a@b.co","password":"x","admin":true}`,
		`{"email":"a@b.co"} {"x":1}`,
		`not json`,
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var in credentials
		if decodeJSON(rec, req, &in) {
			t.Errorf("body %q accepted", body)
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d", body, rec.Code)
		}
	}
}

func TestDecodeJSONAcceptsValidBodyAndRejectsEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.co","password":"x"}`))
	var in credentials
	if !decodeJSON(rec, req, &in) || in.Email != "a@b.co" {
		t.Fatalf("valid body rejected: %+v", in)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(``))
	if decodeJSON(rec, req, &in) {
		t.Fatal("empty body accepted")
	}
}

func TestWriteErrorUsesJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusTeapot, "boom")
	if rec.Code != http.StatusTeapot || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status %d, content type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), `"error":"boom"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

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

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerGeomysOrgFilter(t *testing.T) {
	// Test with geomys.org filter - should return 200 (uptime above 99.5%)
	req := httptest.NewRequest("GET", "/geomys.org", nil)
	w := httptest.NewRecorder()

	handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "geomys.org") {
		t.Errorf("expected body to contain 'geomys.org', got: %s", body)
	}

	// Verify we got some output
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestHandlerDotFilterHighThreshold(t *testing.T) {
	// Test with broad filter and threshold 100 - should fail (503) because
	// at least one entry will be below 100%
	req := httptest.NewRequest("GET", "/http?threshold=100", nil)
	w := httptest.NewRecorder()

	handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	body := w.Body.String()
	// Should have many results since "http" matches most URLs
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestHandlerMissingFilter(t *testing.T) {
	// Test with missing filter - should return 400
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandlerInvalidThreshold(t *testing.T) {
	// Test with invalid threshold - should return 400
	req := httptest.NewRequest("GET", "/geomys.org?threshold=invalid", nil)
	w := httptest.NewRecorder()

	handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "invalid threshold") {
		t.Errorf("expected error message about invalid threshold, got: %s", body)
	}
}

func TestHandlerCustomThreshold(t *testing.T) {
	// Test with a low threshold that should pass
	req := httptest.NewRequest("GET", "/geomys.org?threshold=90", nil)
	w := httptest.NewRecorder()

	handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

package main

import (
	"os"
	"testing"

	"github.com/cespare/webtest"
)

func TestHandler(t *testing.T) {
	if os.Getenv("BUTTONDOWN_API_KEY") == "" {
		t.Skip("BUTTONDOWN_API_KEY not set, skipping handler tests")
	}
	webtest.TestHandler(t, "*_test.txt", handler())
}

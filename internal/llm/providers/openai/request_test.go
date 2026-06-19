package openai

import (
	"net/http"
	"testing"

	"github.com/openai/openai-go/v3/option"
)

func TestOnResponseMiddlewareObservesEachAttempt(t *testing.T) {
	type seen struct {
		status  int
		headers http.Header
	}
	var calls []seen
	mw := onResponseMiddleware(func(status int, headers http.Header) {
		calls = append(calls, seen{status, headers})
	})

	// Simulate the SDK re-running the middleware chain across a retry: first a
	// 429, then a 200. The hook must fire once per attempt.
	attempts := []*http.Response{
		{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"1"}}},
		{StatusCode: http.StatusOK, Header: http.Header{}},
	}
	for _, resp := range attempts {
		next := func(*http.Request) (*http.Response, error) { return resp, nil }
		if _, err := mw(&http.Request{}, option.MiddlewareNext(next)); err != nil {
			t.Fatalf("middleware returned error: %v", err)
		}
	}

	if len(calls) != 2 {
		t.Fatalf("expected hook to fire twice, got %d", len(calls))
	}
	if calls[0].status != http.StatusTooManyRequests || calls[0].headers.Get("Retry-After") != "1" {
		t.Fatalf("first attempt not observed correctly: %+v", calls[0])
	}
	if calls[1].status != http.StatusOK {
		t.Fatalf("second attempt status = %d, want 200", calls[1].status)
	}
}

func TestOnResponseMiddlewareSkipsNilResponse(t *testing.T) {
	called := false
	mw := onResponseMiddleware(func(int, http.Header) { called = true })
	next := func(*http.Request) (*http.Response, error) { return nil, http.ErrServerClosed }
	if _, err := mw(&http.Request{}, option.MiddlewareNext(next)); err == nil {
		t.Fatal("expected error to propagate")
	}
	if called {
		t.Fatal("hook must not fire when there is no response")
	}
}

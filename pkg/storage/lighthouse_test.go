package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetLighthouseFileCtx_OK(t *testing.T) {
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ctx := context.Background()
	b, err := GetLighthouseFileCtx(ctx, srv.URL+"/", "cid123", 500*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "ok" {
		t.Fatalf("got %q want %q", string(b), "ok")
	}
}

func TestGetLighthouseFileCtx_Timeout(t *testing.T) {
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()

	start := time.Now()
	ctx := context.Background()
	_, err := GetLighthouseFileCtx(ctx, srv.URL+"/", "cid123", 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if dur := time.Since(start); dur > 500*time.Millisecond {
		t.Fatalf("took too long: %v", dur)
	}
}

func startHTTPServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "operation not permitted") {
				t.Skip("network operations not permitted in sandbox")
			}
			panic(r)
		}
	}()
	return httptest.NewServer(handler)
}

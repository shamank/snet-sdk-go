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

func TestGetLighthouseFile_Success(t *testing.T) {
	expectedContent := []byte("test content")
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(expectedContent)
	}))
	defer srv.Close()

	content, err := GetLighthouseFile(srv.URL+"/", "QmTest")
	if err != nil {
		t.Fatalf("GetLighthouseFile() error = %v", err)
	}

	if string(content) != string(expectedContent) {
		t.Errorf("got %q, want %q", content, expectedContent)
	}
}

func TestGetLighthouseFile_404NotFound(t *testing.T) {
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()

	_, err := GetLighthouseFile(srv.URL+"/", "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}

func TestGetLighthouseFile_500Error(t *testing.T) {
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	_, err := GetLighthouseFile(srv.URL+"/", "test")
	if err == nil {
		t.Fatal("expected error for 500")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got: %v", err)
	}
}

func TestGetLighthouseFile_EmptyContent(t *testing.T) {
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	content, err := GetLighthouseFile(srv.URL+"/", "empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("expected empty, got %d bytes", len(content))
	}
}

func TestGetLighthouseFile_LargeFile(t *testing.T) {
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(largeContent)
	}))
	defer srv.Close()

	content, err := GetLighthouseFile(srv.URL+"/", "large")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(content) != len(largeContent) {
		t.Errorf("size mismatch: got %d, want %d", len(content), len(largeContent))
	}
}

func TestGetLighthouseFileCtx_ContextCancellation(t *testing.T) {
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := GetLighthouseFileCtx(ctx, srv.URL+"/", "slow", 0)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestGetLighthouseFileCtx_NoTimeout(t *testing.T) {
	srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	}))
	defer srv.Close()

	ctx := context.Background()
	content, err := GetLighthouseFileCtx(ctx, srv.URL+"/", "test", 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if string(content) != "test" {
		t.Errorf("got %q, want %q", content, "test")
	}
}

func TestGetLighthouseFile_RealWorldCIDs(t *testing.T) {
	cids := []string{
		"QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG",
		"bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
	}

	for _, cid := range cids {
		t.Run(cid, func(t *testing.T) {
			srv := startHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, cid) {
					t.Errorf("expected %s in path", cid)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			}))
			defer srv.Close()

			_, err := GetLighthouseFile(srv.URL+"/", cid)
			if err != nil {
				t.Errorf("CID %s failed: %v", cid, err)
			}
		})
	}
}

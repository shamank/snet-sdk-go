package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestParseProtoFiles_TarArchive(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Directory entry should be ignored.
	if err := tw.WriteHeader(&tar.Header{
		Name:     "subdir/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	}); err != nil {
		t.Fatalf("write dir header: %v", err)
	}

	// Regular proto file should be captured.
	protoBody := []byte("syntax = \"proto3\"; message X {}")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "subdir/service.proto",
		Mode:     0o644,
		Size:     int64(len(protoBody)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatalf("write proto header: %v", err)
	}
	if _, err := tw.Write(protoBody); err != nil {
		t.Fatalf("write proto body: %v", err)
	}

	// Non-proto file should be skipped.
	txtBody := []byte("ignore me")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "notes.txt",
		Mode:     0o644,
		Size:     int64(len(txtBody)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatalf("write txt header: %v", err)
	}
	if _, err := tw.Write(txtBody); err != nil {
		t.Fatalf("write txt body: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	protos, err := ParseProtoFiles(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseProtoFiles returned error: %v", err)
	}

	if len(protos) != 1 {
		t.Fatalf("expected 1 proto file, got %d", len(protos))
	}
	if got := protos["subdir/service.proto"]; got != string(protoBody) {
		t.Fatalf("unexpected proto content: %q", got)
	}
}

func TestParseProtoFiles_GzipArchive(t *testing.T) {
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)

	body := []byte("syntax = \"proto3\"; message Ping {}")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "ping.proto",
		Mode:     0o644,
		Size:     int64(len(body)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	var gzBuf bytes.Buffer
	gzw := gzip.NewWriter(&gzBuf)
	if _, err := gzw.Write(tarBuf.Bytes()); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	protos, err := ParseProtoFiles(gzBuf.Bytes())
	if err != nil {
		t.Fatalf("ParseProtoFiles returned error: %v", err)
	}
	if _, ok := protos["ping.proto"]; !ok {
		t.Fatalf("expected ping.proto in result map, got %v", protos)
	}
}

func TestParseProtoFiles_UnknownEntryType(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name:     "symlink",
		Mode:     0o777,
		Typeflag: tar.TypeSymlink,
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	if _, err := ParseProtoFiles(buf.Bytes()); err == nil {
		t.Fatal("expected error for unsupported entry type")
	}
}

func TestIsGzipFile(t *testing.T) {
	gz := []byte{0x1F, 0x8B, 0x08}
	plain := []byte("hello")

	if !isGzipFile(gz) {
		t.Fatal("expected gzip magic to be detected")
	}
	if isGzipFile(plain) {
		t.Fatal("plain bytes reported as gzip")
	}
}

func TestClient_UploadJSON_MarshalError(t *testing.T) {
// Create data that cannot be marshaled to JSON
data := map[string]interface{}{
"channel": make(chan int), // channels cannot be marshaled
}

client := &Client{}

_, err := client.UploadJSON(data)

if err == nil {
t.Fatal("expected error for unmarshalable data")
}
if !containsStr(err.Error(), "failed to marshal JSON") {
t.Fatalf("expected marshal error, got: %v", err)
}
}

func TestClient_UploadJSON_NoFetcher(t *testing.T) {
client := &Client{
ipfsFetcher: nil,
HttpApi:     nil,
}

data := map[string]string{"test": "data"}

_, err := client.UploadJSON(data)

if err == nil {
t.Fatal("expected error when no fetcher configured")
}
}

// Helper to check if error message contains substring
func containsStr(s, substr string) bool {
return len(s) >= len(substr) && indexOfStr(s, substr) >= 0
}

func indexOfStr(s, substr string) int {
for i := 0; i <= len(s)-len(substr); i++ {
if s[i:i+len(substr)] == substr {
return i
}
}
return -1
}

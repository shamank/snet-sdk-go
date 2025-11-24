package grpc

import (
	"testing"
)

func TestGetProtoDescriptorsAndFindMethod(t *testing.T) {
	const protoSrc = `
		syntax = "proto3";
		package demo;
		service Greeter {
			rpc SayHello(HelloRequest) returns (HelloReply) {}
		}
		message HelloRequest { string name = 1; }
		message HelloReply { string message = 1; }
	`

	files := map[string]string{"demo.proto": protoSrc}
	fds, err := getProtoDescriptors(files)
	if err != nil {
		t.Fatalf("getProtoDescriptors returned error: %v", err)
	}
	if len(fds) == 0 {
		t.Fatal("expected non-empty descriptor set")
	}

	fd, method, err := FindMethod(fds, "SayHello")
	if err != nil {
		t.Fatalf("FindMethod returned error: %v", err)
	}
	if string(fd.Package()) != "demo" {
		t.Fatalf("unexpected package: %s", fd.Package())
	}
	if string(method.Parent().Name()) != "Greeter" {
		t.Fatalf("unexpected service name: %s", method.Parent().Name())
	}
}

func TestFindMethod_NotFound(t *testing.T) {
	files := map[string]string{"foo.proto": `
		syntax = "proto3";
		package foo;
		service S { rpc Ping(Req) returns (Resp) {} }
		message Req {}
		message Resp {}
	`}
	fds, err := getProtoDescriptors(files)
	if err != nil {
		t.Fatalf("getProtoDescriptors returned error: %v", err)
	}

	if _, _, err := FindMethod(fds, "Unknown"); err == nil {
		t.Fatal("expected error for missing method")
	}
}

func TestGetProtoDescriptors_InvalidSource(t *testing.T) {
	files := map[string]string{"bad.proto": "syntax = \"proto2\"; message X {"}
	if _, err := getProtoDescriptors(files); err == nil {
		t.Fatal("expected compilation error for invalid proto")
	}
}

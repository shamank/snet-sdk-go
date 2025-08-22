package grpc

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"slices"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FindMethod searches the given compiled proto files for a method with the
// provided simple method name (as declared in the .proto). It iterates over all
// services in all files and returns the file descriptor and method descriptor
// for the first match.
//
// Returns:
//   - protoreflect.FileDescriptor: the file containing the method,
//   - protoreflect.MethodDescriptor: the method descriptor,
//   - error: if the method cannot be found.
func FindMethod(files linker.Files, methodName string) (protoreflect.FileDescriptor, protoreflect.MethodDescriptor, error) {
	for _, file := range files {
		for i := 0; i < file.Services().Len(); i++ {
			service := file.Services().Get(i)
			method := service.Methods().ByName(protoreflect.Name(methodName))
			if method != nil {
				return file, method, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("method %s not found in provided proto files", methodName)
}

// TrainingProtoEmbedded contains the embedded text content of training.proto,
// which is compiled alongside user-provided proto sources at runtime.
//
//go:embed training.proto
var TrainingProtoEmbedded string

// getProtoDescriptors compiles the provided proto sources (filename → content)
// into linker.Files using protocompile. It also injects the embedded
// training.proto into the compilation set and enables standard imports.
//
// Returns a non-nil set of file descriptors or an error if compilation fails.
func getProtoDescriptors(protoFiles map[string]string) (linker.Files, error) {
	protoFiles["training.proto"] = TrainingProtoEmbedded
	accessor := protocompile.SourceAccessorFromMap(protoFiles)
	r := protocompile.WithStandardImports(&protocompile.SourceResolver{Accessor: accessor})
	compiler := protocompile.Compiler{
		Resolver:       r,
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	fds, err := compiler.Compile(context.Background(), slices.Collect(maps.Keys(protoFiles))...)
	if err != nil || fds == nil {
		zap.L().Error("failed to compile proto files", zap.Error(err))
		return nil, fmt.Errorf("failed to compile proto files: %v", err)
	}
	return fds, nil
}

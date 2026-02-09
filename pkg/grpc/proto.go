package grpc

import (
	"archive/zip"
	"context"
	_ "embed"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/singnet/snet-sdk-go/pkg/model"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoManager provides operations for managing protobuf files associated with a service.
type ProtoManager interface {
	// Save writes all proto files to a directory preserving their structure
	Save(dirPath string) error
	// SaveAsZip writes all proto files into a ZIP archive
	SaveAsZip(zipPath string) error
	// Get returns a map of proto filenames to their content
	Get() map[string]string
}

// serviceProtoManager encapsulates the logic for working with proto files.
type serviceProtoManager struct {
	serviceMetadata *model.ServiceMetadata
}

// NewProtoManager creates a new ProtoManager for the given service metadata.
func NewProtoManager(serviceMetadata *model.ServiceMetadata) ProtoManager {
	return &serviceProtoManager{
		serviceMetadata: serviceMetadata,
	}
}

// Save writes all loaded proto files to the specified directory,
// creating subdirectories as needed to preserve the file structure.
// Returns an error if directory creation or file writing fails.
func (pm *serviceProtoManager) Save(dirPath string) error {
	protos := pm.serviceMetadata.ProtoFiles
	if len(protos) == 0 {
		return fmt.Errorf("no proto files to save")
	}

	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for filename, content := range protos {
		fullPath := filepath.Join(dirPath, filename)
		if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create subdirectory for %s: %w", filename, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}
	return nil
}

// SaveAsZip writes all loaded .proto files from the ServiceClient
// into a ZIP archive at the specified path.
//
// Each proto file is added to the archive preserving its filename (including any subdirectories).
//
// Returns an error if the ZIP file cannot be created, or if any file cannot be added or written.
func (pm *serviceProtoManager) SaveAsZip(zipPath string) error {
	protos := pm.serviceMetadata.ProtoFiles
	if len(protos) == 0 {
		return fmt.Errorf("no proto files to save")
	}

	outFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer func(outFile *os.File) {
		err = outFile.Close()
		if err != nil {
			zap.L().Error("failed to close zip file", zap.Error(err))
			return
		}
	}(outFile)

	zipWriter := zip.NewWriter(outFile)
	defer func(zipWriter *zip.Writer) {
		err = zipWriter.Close()
		if err != nil {
			zap.L().Error("failed to close zip writer", zap.Error(err))
		}
	}(zipWriter)

	for filename, content := range protos {
		// Create a file entry in the ZIP archive
		fw, err := zipWriter.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file %s in zip: %w", filename, err)
		}

		if _, err = fw.Write([]byte(content)); err != nil {
			return fmt.Errorf("failed to write content for %s: %w", filename, err)
		}
	}

	zap.L().Info("zip file created", zap.Int("files", len(protos)), zap.String("zip_path", zipPath))

	return nil
}

// Get returns a map of all proto files (filename -> content).
func (pm *serviceProtoManager) Get() map[string]string {
	return pm.serviceMetadata.ProtoFiles
}

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

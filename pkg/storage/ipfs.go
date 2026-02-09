package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	"go.uber.org/zap"

	"io"
	"net/http"
	"time"
)

// ParseProtoFiles extracts .proto files from a tar or tar.gz archive.
//
// The input is inspected for a gzip magic header; if present, it is
// transparently decompressed before reading tar entries. Directory entries
// are ignored (the daemon does not support directories in bundles), and any
// non-.proto regular files are skipped. The returned map preserves the
// filenames (including any subdirectories) as keys, with file contents as
// values.
//
// Returns an error if the archive cannot be read or a file within it cannot
// be processed.
func ParseProtoFiles(compressedFile []byte) (protos map[string]string, err error) {
	var reader io.Reader = bytes.NewReader(compressedFile)

	if isGzipFile(compressedFile) {
		zap.L().Debug("Detected gzip-compressed tar file, decompressing...")
		gzr, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip: %w", err)
		}
		defer func(gzr *gzip.Reader) {
			err = gzr.Close()
			if err != nil {
				zap.L().Error("failed to close gzip reader", zap.Error(err))
			}
		}(gzr)
		reader = gzr // Use the decompressed stream
	}

	tarReader := tar.NewReader(reader)
	protos = make(map[string]string)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			zap.L().Error("Failed to read tar entry", zap.Error(err))
			return nil, err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			zap.L().Warn("Directory found in archive, daemon don't support dirs", zap.String("name", header.Name))
		case tar.TypeReg:
			zap.L().Debug("File found in archive", zap.String("name", header.Name))
			data, err := io.ReadAll(tarReader)
			if err != nil {
				zap.L().Error("Failed to read file from tar", zap.Error(err))
				return nil, err
			}
			if !strings.HasSuffix(header.Name, ".proto") { // ignoring not proto files
				zap.L().Info("Detected not .proto file in archive, skipping", zap.String("name", header.Name))
				continue
			}
			protos[header.Name] = string(data)
		default:
			err = fmt.Errorf("unknown file type %c in file %s", header.Typeflag, header.Name)
			zap.L().Error(err.Error())
			return nil, err
		}
	}
	return protos, nil
}

// isGzipFile reports whether data appears to be gzip-compressed,
// based on the 0x1F 0x8B magic bytes.
func isGzipFile(data []byte) bool {
	// Gzip files start with the bytes 0x1F 0x8B
	return len(data) > 2 && data[0] == 0x1F && data[1] == 0x8B
}

// ipfsFetcher is the concrete implementation of IPFSFetcher using Kubo HTTP API.
type ipfsFetcher struct {
	api *rpc.HttpApi
}

// newIPFSFetcher creates a new IPFS fetcher with the given HTTP API client.
func newIPFSFetcher(api *rpc.HttpApi) IPFSFetcher {
	return &ipfsFetcher{api: api}
}

// Fetch content by CID from IPFS using the configured
// Kubo HTTP API client. The supplied hash is normalized via formatHash,
// parsed as a CID, and retrieved via `ipfs cat`. The method then performs
// a best-effort verification by recomputing a CID from (original CID bytes +
// content) and comparing it with the requested CID.
//
// On success, it returns the file contents.
func (f *ipfsFetcher) Fetch(ctx context.Context, hash string) (content []byte, err error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
	}

	hash = formatHash(hash)

	zap.L().Debug("Hash Used to retrieve from IPFS", zap.String("hash", hash))

	if f.api == nil {
		return nil, fmt.Errorf("ipfs client not configured")
	}

	cID, err := cid.Parse(hash)
	if err != nil {
		zap.L().Error("error parsing the ipfs hash", zap.String("hashFromMetaData", hash), zap.Error(err))
	}

	req := f.api.Request("cat", cID.String())
	if err != nil {
		zap.L().Error("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}
	resp, err := req.Send(ctx)
	if err != nil {
		zap.L().Error("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}
	defer func(resp *rpc.Response) {
		err = resp.Close()
		if err != nil {
			zap.L().Error("error closing response in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		}
	}(resp)

	if resp.Error != nil {
		zap.L().Error("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}
	fileContent, err := io.ReadAll(resp.Output)
	if err != nil {
		zap.L().Error("error: in Reading the meta data file", zap.Error(err), zap.String("hashFromMetaData", hash))
		return
	}

	// Create a CID manually to check CID equivalence.
	_, c, err := cid.CidFromBytes(append(cID.Bytes(), fileContent...))
	if err != nil {
		zap.L().Error("error generating ipfs hash", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}

	// To test if two CIDs are equivalent, be sure to use the 'Equals' method:
	if !c.Equals(cID) {
		zap.L().Error("IPFS hash verification failed. Generated hash does not match with expected hash",
			zap.String("expectedHash", hash),
			zap.String("hashFromIPFSContent", c.String()))
	}

	return fileContent, err
}

// GetFileFromIPFS fetches content by CID from IPFS using the configured
// backend fetcher. It is kept for backward compatibility.
func (c *Client) GetFileFromIPFS(ctx context.Context, hash string) ([]byte, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
	}

	if c.ipfsFetcher == nil {
		c.ipfsFetcher = newIPFSFetcher(c.HttpApi)
	}
	return c.ipfsFetcher.Fetch(ctx, hash)
}

// UploadJSON serializes data to JSON and uploads it to IPFS.
// Returns the IPFS URI (ipfs://<hash>) on success.
func (c *Client) UploadJSON(ctx context.Context, data interface{}) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		zap.L().Error("error marshaling data to json", zap.Error(err))
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if c.ipfsFetcher == nil {
		c.ipfsFetcher = newIPFSFetcher(c.HttpApi)
	}

	// Cast to access Upload method (we'll need to add this to the interface)
	uploader, ok := c.ipfsFetcher.(*ipfsFetcher)
	if !ok {
		return "", fmt.Errorf("ipfs fetcher does not support uploads")
	}

	return uploader.Upload(ctx, jsonData)
}

// Upload uploads data to IPFS and returns the IPFS URI (ipfs://<hash>).
// The data is added using the IPFS HTTP API 'add' command.
func (f *ipfsFetcher) Upload(ctx context.Context, data []byte) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
	}

	if f.api == nil {
		return "", fmt.Errorf("ipfs client not configured")
	}

	reader := bytes.NewReader(data)
	req := f.api.Request("add")
	req.Body(reader)

	resp, err := req.Send(ctx)
	if err != nil {
		zap.L().Error("error uploading to ipfs", zap.Error(err))
		return "", err
	}
	defer func(resp *rpc.Response) {
		err = resp.Close()
		if err != nil {
			zap.L().Error("error closing ipfs response", zap.Error(err))
		}
	}(resp)

	if resp.Error != nil {
		zap.L().Error("ipfs add command returned error", zap.Error(resp.Error))
		return "", resp.Error
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Output)
	if err != nil {
		zap.L().Error("error reading ipfs add response", zap.Error(err))
		return "", err
	}

	var addResp struct {
		Hash string `json:"Hash"`
	}
	if err := json.Unmarshal(body, &addResp); err != nil {
		zap.L().Error("error unmarshaling ipfs add response", zap.Error(err))
		return "", err
	}

	zap.L().Debug("Successfully uploaded to IPFS", zap.String("hash", addResp.Hash))
	return IpfsPrefix + addResp.Hash, nil
}

// NewIPFSClient constructs a Kubo HTTP API client pointed at url.
// The client uses a short HTTP timeout suitable for metadata and API-source downloads.
func NewIPFSClient(url string) (client *rpc.HttpApi, err error) {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}
	client, err = rpc.NewURLApiWithClient(url, &httpClient)
	if err != nil {
		zap.L().Panic("Connection failed to IPFS", zap.String("url", url), zap.Error(err))
	}
	return client, err
}

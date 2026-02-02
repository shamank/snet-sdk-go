// Package storage provides helpers to retrieve artifacts (e.g., metadata,
// .proto bundles) from decentralized storage backends used by SingularityNET.
// Currently supported sources are IPFS (via a Kubo HTTP API client) and the
// Lighthouse gateway for Filecoin content.
package storage

import (
	"regexp"
	"strings"

	"github.com/ipfs/kubo/client/rpc"
	"go.uber.org/zap"
)

const (
	// IpfsPrefix is the URI scheme prefix recognized for IPFS content.
	IpfsPrefix = "ipfs://"
	// FilecoinPrefix is the URI scheme prefix recognized for Filecoin/Lighthouse content.
	FilecoinPrefix = "filecoin://"
)

// Storage is a minimal interface for backends able to fetch and store blobs by ID/hash.
type Storage interface {
	ReadFile(id string) ([]byte, error)
	UploadJSON(data interface{}) (string, error)
}

// LighthouseFetcher fetches content from a Lighthouse gateway.
type LighthouseFetcher interface {
	Fetch(endpoint, cid string) ([]byte, error)
}

// IPFSFetcher fetches content addressed by CID from IPFS.
type IPFSFetcher interface {
	Fetch(hash string) ([]byte, error)
}

// Client aggregates the configured storage backends.
//
// Note: The field name LighthouseUrl is kept for backward compatibility,
// even though the idiomatic Go name would be LighthouseURL.
type Client struct {
	// HttpApi is a connected Kubo HTTP API client used for IPFS reads.
	*rpc.HttpApi
	// LighthouseUrl is the base URL of the Lighthouse HTTP gateway.
	LighthouseUrl string

	lighthouseFetcher LighthouseFetcher
	ipfsFetcher       IPFSFetcher
}

// NewStorage constructs a Storage helper using the provided IPFS API endpoint
// and Lighthouse gateway URL. If the IPFS client fails to initialize, the error
// is logged and the returned struct may have a nil HttpApi.
func NewStorage(ipfsURL, lighthouseURL string) *Client {
	var err error
	s := new(Client)
	s.HttpApi, err = NewIPFSClient(ipfsURL)
	s.LighthouseUrl = lighthouseURL
	s.lighthouseFetcher = defaultLighthouseFetcher{}
	s.ipfsFetcher = newIPFSFetcher(s.HttpApi)
	if err != nil {
		zap.L().Error(err.Error())
	}
	return s
}

// ReadFile fetches content identified by the given hash/URI. If the input has
// the "filecoin://" prefix, it is retrieved via the Lighthouse gateway;
// otherwise, the content is fetched from IPFS using the Kubo client.
// The hash/URI is normalized with formatHash before retrieval.
func (s *Client) ReadFile(hash string) (rawFile []byte, err error) {
	if s.lighthouseFetcher == nil {
		s.lighthouseFetcher = defaultLighthouseFetcher{}
	}
	if s.ipfsFetcher == nil {
		s.ipfsFetcher = newIPFSFetcher(s.HttpApi)
	}

	if strings.HasPrefix(hash, FilecoinPrefix) {
		rawFile, err = s.lighthouseFetcher.Fetch(s.LighthouseUrl, formatHash(hash))
	} else {
		rawFile, err = s.ipfsFetcher.Fetch(formatHash(hash))
	}
	return rawFile, err
}

// defaultLighthouseFetcher is the production implementation of LighthouseFetcher.
// It uses the real HTTP client to fetch content from Lighthouse gateway.
type defaultLighthouseFetcher struct{}

func (defaultLighthouseFetcher) Fetch(endpoint, cid string) ([]byte, error) {
	return GetLighthouseFile(endpoint, cid)
}

// formatHash removes known URI scheme prefixes and any non-alphanumeric
// characters (except '=') from the supplied hash/URI to produce a clean CID
// string suitable for the underlying backends.
func formatHash(hash string) string {
	hash = strings.Replace(hash, IpfsPrefix, "", -1)
	hash = strings.Replace(hash, FilecoinPrefix, "", -1)
	hash = removeSpecialCharacters(hash)
	return hash
}

// removeSpecialCharacters strips all characters except ASCII letters, digits,
// and '=' from pString. Used to sanitize incoming CIDs/IDs.
func removeSpecialCharacters(pString string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9=]")
	return reg.ReplaceAllString(pString, "")
}

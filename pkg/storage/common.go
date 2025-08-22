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

// storage is a minimal interface for backends able to fetch a blob by ID/hash.
type storage interface {
	ReadFile(id string) ([]byte, error)
}

// Storage aggregates the configured storage backends.
//
// Note: The field name LighthouseUrl is kept for backward compatibility,
// even though the idiomatic Go name would be LighthouseURL.
type Storage struct {
	// HttpApi is a connected Kubo HTTP API client used for IPFS reads.
	*rpc.HttpApi
	// LighthouseUrl is the base URL of the Lighthouse HTTP gateway.
	LighthouseUrl string
}

// NewStorage constructs a Storage helper using the provided IPFS API endpoint
// and Lighthouse gateway URL. If the IPFS client fails to initialize, the error
// is logged and the returned struct may have a nil HttpApi.
func NewStorage(ipfsURL, lighthouseURL string) *Storage {
	var err error
	s := new(Storage)
	s.HttpApi, err = NewIPFSClient(ipfsURL)
	s.LighthouseUrl = lighthouseURL
	if err != nil {
		zap.L().Error(err.Error())
	}
	return s
}

// ReadFile fetches content identified by the given hash/URI. If the input has
// the "filecoin://" prefix, it is retrieved via the Lighthouse gateway;
// otherwise, the content is fetched from IPFS using the Kubo client.
// The hash/URI is normalized with formatHash before retrieval.
func (s *Storage) ReadFile(hash string) (rawFile []byte, err error) {
	if strings.HasPrefix(hash, FilecoinPrefix) {
		rawFile, err = GetLighthouseFile(s.LighthouseUrl, formatHash(hash))
	} else {
		rawFile, err = s.GetFileFromIPFS(formatHash(hash))
	}
	return rawFile, err
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

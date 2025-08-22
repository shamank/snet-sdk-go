// Package grpc provides a lightweight dynamic gRPC client that can invoke RPC
// methods without generated stubs. It compiles provided .proto sources at
// runtime (via protocompile) and uses dynamicpb to marshal/unmarshal requests
// and responses. Calls can be made with native proto messages, JSON payloads,
// or plain Go maps.
package grpc

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bufbuild/protocompile/linker"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Client is a dynamic gRPC client that holds a connected gRPC ClientConn and a
// set of compiled file descriptors used to locate services/methods at runtime.
type Client struct {
	// GRPC is the underlying client connection.
	GRPC *grpc.ClientConn `json:"-"`
	// ProtoFiles are the compiled descriptors of the provided .proto sources.
	ProtoFiles linker.Files `json:"-"`
}

// NewClient creates a dynamic gRPC client for the given endpoint and set of
// .proto files (as filename → file content). The endpoint scheme determines
// transport security:
//   - "https://": TLS (system defaults)
//   - "http://":  insecure
//   - no scheme:  insecure
//
// The provided proto files are compiled at runtime; if compilation fails the
// connection is closed and nil is returned. The returned client proactively
// starts connecting (ClientConn.Connect()).
func NewClient(endpoint string, protoFiles map[string]string) *Client {
	addr, creds := grpcCredsFromEndpoint(endpoint)
	conn, err := grpc.NewClient(addr, creds)
	if err != nil {
		zap.L().Error(err.Error())
		return nil
	}

	descriptors, err := getProtoDescriptors(protoFiles)
	if err != nil {
		_ = conn.Close()
		return nil
	}

	conn.Connect()

	return &Client{
		GRPC:       conn,
		ProtoFiles: descriptors,
	}
}

// Close shuts down the underlying gRPC connection.
// It is safe to call on a nil receiver or when GRPC is nil.
func (c *Client) Close() error {
	if c == nil || c.GRPC == nil {
		return nil
	}
	return c.GRPC.Close()
}

// CallWithMap invokes a unary RPC by method name using a map as the request
// body. The map is JSON-encoded and then routed through CallWithJSON.
// Method should be the simple method name as declared in the .proto (not the
// fully-qualified path).
func (c *Client) CallWithMap(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	jsonStr, err := c.CallWithJSON(ctx, method, jsonData)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonStr, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// CallWithProto invokes a unary RPC by method name with a concrete proto.Message
// request and returns a dynamic proto.Message response.
// The method is resolved via the in-memory descriptors; the final fully-qualified
// method path is built as "/<package>.<Service>/<Method>".
func (c *Client) CallWithProto(ctx context.Context, method string, req proto.Message) (proto.Message, error) {
	fd, methodDesc, err := FindMethod(c.ProtoFiles, method)
	if err != nil {
		return nil, err
	}
	out := dynamicpb.NewMessage(methodDesc.Output())
	err = c.GRPC.Invoke(ctx, "/"+string(fd.Package())+"."+string(methodDesc.Parent().Name())+"/"+method, req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CallWithJSON invokes a unary RPC by method name using a JSON request body.
// The JSON is unmarshalled into a dynamic input message (discarding unknown
// fields and allowing partial messages), the call is performed, and the
// response is marshaled back to JSON with proto field names and unpopulated
// fields emitted.
func (c *Client) CallWithJSON(ctx context.Context, method string, body []byte) ([]byte, error) {
	fd, methodDesc, err := FindMethod(c.ProtoFiles, method)
	if err != nil {
		return nil, err
	}

	in := dynamicpb.NewMessage(methodDesc.Input())
	out := dynamicpb.NewMessage(methodDesc.Output())

	err = protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}.Unmarshal(body, in)
	if err != nil {
		return nil, err
	}

	fullMethod := "/" + string(fd.Package()) + "." + string(methodDesc.Parent().Name()) + "/" + method
	err = c.GRPC.Invoke(ctx, fullMethod, in, out)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}.Marshal(out)
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

// grpcCredsFromEndpoint derives a dial address and dial option from an endpoint URL.
// "https://" enables TLS; "http://" and bare addresses use insecure credentials.
func grpcCredsFromEndpoint(endpoint string) (string, grpc.DialOption) {
	if strings.HasPrefix(endpoint, "https://") {
		return strings.TrimPrefix(endpoint, "https://"), grpc.WithTransportCredentials(credentials.NewTLS(nil))
	}
	if strings.HasPrefix(endpoint, "http://") {
		return strings.TrimPrefix(endpoint, "http://"), grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	return endpoint, grpc.WithTransportCredentials(insecure.NewCredentials())
}

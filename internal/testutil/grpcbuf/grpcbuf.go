package grpcbuf

import (
	"context"
	"net"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

const bufSize = 1024 * 1024

// MetaCapture captures incoming metadata on the server side for later inspection in tests.
type MetaCapture struct {
	last atomic.Value // stores metadata.MD
}

// Interceptor records incoming metadata and forwards the request to the next handler.
func (m *MetaCapture) Interceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		m.last.Store(md)
	}
	return handler(ctx, req)
}

// Last returns the most recently captured metadata or nil if none.
func (m *MetaCapture) Last() metadata.MD {
	if v := m.last.Load(); v != nil {
		return v.(metadata.MD)
	}
	return nil
}

// EchoServer defines a minimal echo service used in tests.
type EchoServer interface {
	Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
}

type echoServer struct{}

func (s *echoServer) Ping(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func _Echo_Ping_Handler(
	srv interface{},
	ctx context.Context,
	dec func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/test.Echo/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoServer).Ping(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// EchoServiceDesc describes the in-memory echo service used by grpcbuf helpers.
var EchoServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.Echo",
	HandlerType: (*EchoServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Ping", Handler: _Echo_Ping_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "echo_test",
}

// StartServer spins up a bufconn-backed gRPC server with metadata capture enabled.
func StartServer() (*grpc.Server, *bufconn.Listener, *MetaCapture) {
	lis := bufconn.Listen(bufSize)
	cap := &MetaCapture{}
	srv := grpc.NewServer(grpc.UnaryInterceptor(cap.Interceptor))
	srv.RegisterService(&EchoServiceDesc, &echoServer{})
	go func() { _ = srv.Serve(lis) }()
	return srv, lis, cap
}

// Dial connects to the provided bufconn listener using the standard gRPC client stack.
func Dial(ctx context.Context, lis *bufconn.Listener, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	// Use insecure credentials because bufconn does not provide TLS.
	// Use NewClient with a passthrough target so the custom dialer is honored.
	base := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
	}
	base = append(base, opts...)
	return grpc.NewClient("passthrough://bufnet", base...)
}

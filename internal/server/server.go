package server

import (
	"context"

	api "github.com/arindas/proglog/api/log_v1"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Configuration for server. Contains providers for different services.
type Config struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

type grpcServer struct {
	api.UnimplementedLogServer
	*Config
}

const (
	objectWildCard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

var _ api.LogServer = (*grpcServer)(nil)

func newgrpcServer(config *Config) (*grpcServer, error) {
	return &grpcServer{Config: config}, nil
}

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(codes.Unknown, "couldn't find peer info").Err()
	}

	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

type subjectContextKey struct{}

func subject(ctx context.Context) string { return ctx.Value(subjectContextKey{}).(string) }

// Produces a single record to the log which backs this server.
func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildCard, produceAction); err != nil {
		return nil, err
	}

	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

// Consumes the record present at the given offset from the log which backs this server.
func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildCard, consumeAction); err != nil {
		return nil, err
	}

	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{Record: record}, nil
}

// Bidrectional endpoint, reads records from the given stream, produces them to the underlying commit log
// and writes the offsets written to, to the given stream.
func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}

		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

// Sequentially reads the records from the given offset to the end of the log and writes then to
// the given stream.
func (s *grpcServer) ConsumeStream(req *api.ConsumeRequest, stream api.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}

			if err = stream.Send(res); err != nil {
				return err
			}

			req.Offset++
		}
	}
}

// Creates a new GRPC server for our log service with the given configuration.
func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	opts = append(opts,
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(grpc_auth.StreamServerInterceptor(authenticate))),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(grpc_auth.UnaryServerInterceptor(authenticate))),
	)

	gsrv := grpc.NewServer(opts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}

	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}

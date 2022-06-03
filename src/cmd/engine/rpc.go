package engine

import (
	"fmt"
	"net"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/onflow/flow-go/engine"
	"github.com/onflow/flow-go/network"
	jsoncodec "github.com/onflow/flow-go/network/codec/json"
	"github.com/onflow/flow-go/utils/grpcutils"
	accessproto "github.com/onflow/flow/protobuf/go/flow/access"
)

// Config defines the configurable options for the gRPC server.
type Config struct {
	ListenAddr string
	MaxMsgSize int // In bytes
}

// RPC implements a gRPC server for the API service node
// The RPC proxy reads from the channel and forwards requests to the gRPC clients
type RPC struct {
	unit   *engine.Unit
	log    zerolog.Logger
	server *grpc.Server // the gRPC server
	config Config
	codec  network.Codec

	proxy accessproto.AccessAPIServer
}

// New returns a new RPC engine.
func New(log zerolog.Logger, config Config, proxy accessproto.AccessAPIServer) (*RPC, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy argument not set")
	}

	log = log.With().Str("engine", "rpc").Logger()

	codec := jsoncodec.NewCodec()

	if config.MaxMsgSize == 0 {
		config.MaxMsgSize = grpcutils.DefaultMaxMsgSize
	}

	eng := &RPC{
		log:  log,
		unit: engine.NewUnit(),
		server: grpc.NewServer(
			grpc.MaxRecvMsgSize(config.MaxMsgSize),
			grpc.MaxSendMsgSize(config.MaxMsgSize),
		),
		config: config,
		codec:  codec,
		proxy:  proxy,
	}

	accessproto.RegisterAccessAPIServer(eng.server, proxy)

	return eng, nil
}

// Ready returns a ready channel that is closed once the engine has fully
// started. The RPC engine is ready when the gRPC server has successfully
// started.
func (e *RPC) Ready() <-chan struct{} {
	e.unit.Launch(e.serve)
	return e.unit.Ready()
}

// Done returns a done channel that is closed once the engine has fully stopped.
// It sends a signal to stop the gRPC server, then closes the channel.
func (e *RPC) Done() <-chan struct{} {
	return e.unit.Done(e.server.GracefulStop)
}

// serve starts the gRPC server .
//
// When this function returns, the server is considered ready.
func (e *RPC) serve() {
	e.log.Info().Msgf("starting server on address %s", e.config.ListenAddr)

	l, err := net.Listen("tcp", e.config.ListenAddr)
	if err != nil {
		e.log.Err(err).Msg("failed to start server")
		return
	}

	err = e.server.Serve(l)
	if err != nil {
		e.log.Err(err).Msg("fatal error in server")
	}
}

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
		return nil, fmt.Errorf("proxy not set")
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

/*
// SubmitLocal submits an event originating on the local node.
func (e *RPC) SubmitLocal(event interface{}) {
	e.unit.Launch(func() {
		err := e.process(e.me.NodeID(), event)
		if err != nil {
			e.log.Error().Err(err).Msg("could not process submitted event")
		}
	})
}

// Submit submits the given event from the node with the given origin ID
// for processing in a non-blocking manner. It returns instantly and logs
// a potential processing error internally when done.
func (e *RPC) Submit(channel network.Channel, originID flow.Identifier, event interface{}) {
	e.unit.Launch(func() {
		err := e.process(originID, event)
		if err != nil {
			e.log.Error().Err(err).Msg("could not process submitted event")
		}
	})
}

// ProcessLocal processes an event originating on the local node.
func (e *RPC) ProcessLocal(event interface{}) error {
	return e.unit.Do(func() error {
		return e.process(e.me.NodeID(), event)
	})
}

// Process processes the given event from the node with the given origin ID in
// a blocking manner. It returns the potential processing error when done.
func (e *RPC) Process(channel network.Channel, originID flow.Identifier, event interface{}) error {
	return e.unit.Do(func() error {
		return e.process(originID, event)
	})
}

func (e *RPC) process(originID flow.Identifier, event interface{}) error {

	// json encode the message into bytes
	encodedMsg, err := e.codec.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode message: %v", err)
	}

	// create a protobuf message
	flowMessage := ghost.FlowMessage{
		SenderID: originID[:],
		Message:  encodedMsg,
	}

	// write it to the channel
	select {
	case e.messages <- flowMessage:
	default:
		return fmt.Errorf("dropping message since queue is full: %v", err)
	}
	return nil
}
*/
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

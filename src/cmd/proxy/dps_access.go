package proxy

import (
	"errors"
	"net"
	"os"
	"time"

	flowDpsAccess "github.com/GetElastech/flow-dps-access/api"
	dpsApi "github.com/GetElastech/flow-dps/api/dps"
	"google.golang.org/grpc"

	"github.com/GetElastech/flow-dps/codec/zbor"
	"github.com/GetElastech/flow-dps/service/invoker"
	"github.com/rs/zerolog"
)

func NewDpsAccessServer(flowDpsHostUrl string, flowDpsListenPort string, flowDpsMaxCacheSize uint64) (*flowDpsAccess.Server, net.Listener, error) {
	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)

	//Initialize codec.
	codec := zbor.NewCodec()

	//Initialize the API client.
	conn, err := grpc.Dial(flowDpsHostUrl, grpc.WithInsecure())
	if err != nil {
		log.Error().Str("dps", flowDpsHostUrl).Err(err).Msg("could not dial API host")
		return nil, nil, errors.New("Failed to initialize grpc client connection")
	}
	defer conn.Close()

	client := dpsApi.NewAPIClient(conn)
	index := dpsApi.IndexFromAPI(client, codec)

	invoke, err := invoker.New(index, invoker.WithCacheSize(flowDpsMaxCacheSize))
	if err != nil {
		log.Error().Err(err).Msg("could not initialize script invoker")
		return nil, nil, errors.New("error initializing script invoker")
	}

	listener, err := net.Listen("tcp", flowDpsListenPort)
	if err != nil {
		log.Error().Str("address", flowDpsListenPort).Err(err).Msg("could not listen")
		return nil, nil, errors.New("Failed to initialize listener")
	}

	flowDpsAccessServer := flowDpsAccess.NewServer(index, codec, invoke)
	if err != nil {
		log.Error().Str("address", flowDpsListenPort).Err(err).Msg("could not listen")
		return nil, nil, errors.New("Failed to initialize listener")
	}

	return flowDpsAccessServer, listener, nil
}

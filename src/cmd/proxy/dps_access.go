package proxy

import (
	"errors"
	"os"
	"time"

	flowDpsAccess "github.com/GetElastech/flow-dps-access/api"
	dpsApi "github.com/GetElastech/flow-dps/api/dps"
	"google.golang.org/grpc"

	"github.com/GetElastech/flow-dps/codec/zbor"
	"github.com/GetElastech/flow-dps/service/invoker"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/engine/access/rpc/backend"

	"google.golang.org/grpc/credentials"

	"crypto/tls"
)

func NewDpsAccessServer(flowDpsHostUrl string, flowDpsMaxCacheSize uint64, useSecure bool, tlsConfig *tls.Config, timeout time.Duration) (*flowDpsAccess.Server, error) {
	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)

	//Initialize codec.
	codec := zbor.NewCodec()

	if useSecure {
		//Initialize the API client.
		conn, err := grpc.Dial(flowDpsHostUrl, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), backend.WithClientUnaryInterceptor(timeout))
		if err != nil {
			log.Error().Str("dps", flowDpsHostUrl).Err(err).Msg("could not dial API host")
			return nil, errors.New("Failed to initialize grpc client connection")
		}
		defer conn.Close()

		client := dpsApi.NewAPIClient(conn)
		index := dpsApi.IndexFromAPI(client, codec)

		invoke, err := invoker.New(index, invoker.WithCacheSize(flowDpsMaxCacheSize))
		if err != nil {
			log.Error().Err(err).Msg("could not initialize script invoker")
			return nil, errors.New("error initializing script invoker")
		}

		flowDpsAccessServer := flowDpsAccess.NewServer(index, codec, invoke)

		return flowDpsAccessServer, nil
	} else {
		// The blockchain network is by design secure up to the level adjusted by its operators.
		// The connection is insecure meaning the traffic may be watched and tampered with.
		// The blockchain network provides measures to mitigate the latter.

		//Initialize the API client.
		conn, err := grpc.Dial(flowDpsHostUrl, grpc.WithInsecure(), backend.WithClientUnaryInterceptor(timeout))
		if err != nil {
			log.Error().Str("dps", flowDpsHostUrl).Err(err).Msg("could not dial API host")
			return nil, errors.New("Failed to initialize grpc client connection")
		}
		defer conn.Close()

		client := dpsApi.NewAPIClient(conn)
		index := dpsApi.IndexFromAPI(client, codec)

		invoke, err := invoker.New(index, invoker.WithCacheSize(flowDpsMaxCacheSize))
		if err != nil {
			log.Error().Err(err).Msg("could not initialize script invoker")
			return nil, errors.New("error initializing script invoker")
		}

		flowDpsAccessServer := flowDpsAccess.NewServer(index, codec, invoke)

		return flowDpsAccessServer, nil
	}
}

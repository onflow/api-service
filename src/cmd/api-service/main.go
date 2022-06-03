package main

import (
	"fmt"
	"github.com/onflow/api-service/m/v2/cmd/api-service/builder"
	"github.com/onflow/api-service/m/v2/cmd/engine"
	"github.com/onflow/api-service/m/v2/cmd/proxy"
	srv "github.com/onflow/api-service/m/v2/cmd/service"
)

func main() {
	serviceBuilder := builder.NewFlowAPIServiceBuilder()

	// parse all the command line args
	if err := serviceBuilder.Initialize(); err != nil {
		serviceBuilder.ServiceConfig.Logger.Fatal().Err(err).Send()
	}

	// build dependencies and behavior
	serviceBuilder.
		Module("API Service", func(node *srv.ServiceConfig) error {
			ids, err := serviceBuilder.BootstrapIdentities(serviceBuilder.UpstreamNodeAddresses, serviceBuilder.UpstreamNodePublicKeys)
			if err != nil {
				serviceBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			for _, id := range ids {
				serviceBuilder.ServiceConfig.Logger.Info().Str("Upstream access/observer", id.Address).Msg("API Service client")
			}

			serviceBuilder.Api, err = proxy.NewFlowAPIService(ids, serviceBuilder.ApiTimeout)
			if err != nil {
				serviceBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}
			serviceBuilder.ServiceConfig.Logger.Info().Str("cmd", fmt.Sprintf("%v", serviceBuilder.UpstreamNodeAddresses)).Msg("API Service started")
			return nil
		}).
		Module("RPC Engine", func(node *srv.ServiceConfig) error {
			rpcEng, err := engine.New(node.Logger, serviceBuilder.RpcConf, serviceBuilder.Api)
			if err != nil {
				serviceBuilder.ServiceConfig.Logger.Info().Err(err)
				return err
			}

			serviceBuilder.RpcEngine = rpcEng
			serviceBuilder.ServiceConfig.Logger.Info().Str("module", node.Name).Msg("RPC engine started")
			return nil
		}).
		Component("RPC Listening", func(node *srv.ServiceConfig) error {
			// wait until started
			<-serviceBuilder.RpcEngine.Ready()
			serviceBuilder.ServiceConfig.Logger.Info().Msg("Flow API Service Ready")
			return nil
		})

	service, err := serviceBuilder.Build()
	if err != nil {
		serviceBuilder.ServiceConfig.Logger.Err(err)
	}

	err = service.Run()
	if err != nil {
		serviceBuilder.ServiceConfig.Logger.Err(err)
	}
}

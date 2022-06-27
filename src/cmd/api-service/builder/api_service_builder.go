package builder

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/onflow/api-service/m/v2/cmd/engine"
	"github.com/onflow/api-service/m/v2/cmd/service"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"
)

func NewFlowAPIServiceBuilder() *FlowAPIServiceBuilder {
	return &FlowAPIServiceBuilder{
		FlowServiceBuilder: service.NewFlowServiceBuilder("api-service"),
	}
}

type FlowAPIServiceCmd struct {
	service.FlowService
	ServiceConfig service.ServiceConfig
	RpcEngine     *engine.RPC
}

type FlowAPIServiceBuilder struct {
	*service.FlowServiceBuilder
	RpcConf                 engine.Config
	ApiTimeout              time.Duration
	ProtocolNodeAddresses   []string
	ProtocolNodePublicKeys  []string
	ExecutionNodeAddresses  []string
	ExecutionNodePublicKeys []string
	FlowDpsNodeAddress      string
	FlowDpsHostUrl          string
	FlowDpsMaxCacheSize     uint64
	FlosDpsFlagLevel        string
	Api                     access.AccessAPIServer
	RpcEngine               *engine.RPC
}

// Initialize Parse and print API Service command line arguments.
func (fsb *FlowAPIServiceBuilder) Initialize() error {
	flags := &fsb.ServiceConfig.Flags
	flags.StringVarP(&fsb.RpcConf.ListenAddr, "rpc-addr", "r", ":9000", "the address the GRPC server listens on")
	flags.DurationVar(&fsb.ApiTimeout, "flow-api-timeout", 3*time.Second, "tcp timeout of the Flow API gRPC socket")
	flags.StringSliceVar(&fsb.ProtocolNodeAddresses, "protocol-node-addresses", []string{}, "the network addresses of the bootstrap access nodes e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653")
	flags.StringSliceVar(&fsb.ProtocolNodePublicKeys, "protocol-node-public-keys", []string{}, "the networking public key of the bootstrap access nodes (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\"")
	flags.StringSliceVar(&fsb.ExecutionNodeAddresses, "execution-node-addresses", []string{}, "the network addresses of the bootstrap access nodes e.g. access-001.mainnet.flow.org:9653,access-002.mainnet.flow.org:9653")
	flags.StringSliceVar(&fsb.ExecutionNodePublicKeys, "execution-node-public-keys", []string{}, "the networking public key of the bootstrap access nodes (in the same order as the bootstrap node addresses) e.g. \"d57a5e9c5.....\",\"44ded42d....\"")
	flags.StringVarP(&fsb.FlowDpsNodeAddress, "flow-dps-node-address", "a", "127.0.0.1:5006", "address to serve Access API on")
	flags.StringVarP(&fsb.FlowDpsHostUrl, "flow-dps-host-url", "d", "127.0.0.1:5005", "host URL for Flow DPS API endpoints")
	flags.Uint64Var(&fsb.FlowDpsMaxCacheSize, "cache-size", 1_000_000_000, "maximum cache size for register reads in flow-dps in bytes")
	flags.StringVarP(&fsb.FlosDpsFlagLevel, "level", "l", "info", "log output level")

	// This one just prints the flags
	err := fsb.FlowServiceBuilder.ParseAndPrintFlags()
	if err != nil {
		fsb.ServiceConfig.Logger.Fatal().Err(err)
	}

	fsb.ServiceConfig.Logger.Info().
		Str("protocol-node-addresses", fmt.Sprintf("%v", fsb.ProtocolNodeAddresses)).
		Str("protocol-node-public-keys", fmt.Sprintf("%v", fsb.ProtocolNodePublicKeys))
	fsb.ServiceConfig.Logger.Info().
		Str("execution-node-addresses", fmt.Sprintf("%v", fsb.ExecutionNodeAddresses)).
		Str("execution-node-public-keys", fmt.Sprintf("%v", fsb.ExecutionNodePublicKeys))
	fsb.ServiceConfig.Logger.Info().
		Str("flow-dps-node-address", fmt.Sprintf("%v", fsb.FlowDpsNodeAddress)).
		Str("flow-dps-host-url", fmt.Sprintf("%v", fsb.FlowDpsHostUrl))

	return nil
}

// Build a generic service and add API service extras
func (fsb *FlowAPIServiceBuilder) Build() (*FlowAPIServiceCmd, error) {
	fs, err := fsb.FlowServiceBuilder.Build()
	if err != nil {
		return nil, err
	}

	return &FlowAPIServiceCmd{
		FlowService:   *fs,
		ServiceConfig: fsb.ServiceConfig,
		RpcEngine:     fsb.RpcEngine,
	}, nil
}

func (fsb *FlowAPIServiceCmd) Run() error {
	// 1: Start up
	// Start all the components
	err := fsb.ServiceConfig.Start()
	if err != nil {
		return err
	}

	// 2: Listen to SIGINT
	// Graceful shutdown on SIGINT
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT)
	defer signal.Stop(sigint)
	<-sigint

	// 3: Shutdown
	// Shut down the service
	fsb.ServiceConfig.Logger.Info().Msg("Flow API Service Done")
	<-fsb.RpcEngine.Done()

	return nil
}

// BootstrapIdentities converts the bootstrap node addresses and keys to a Flow Identity list where
// each Flow Identity is initialized with the passed address, the networking key
// and the Node ID set to ZeroID, role set to Access, 0 stake and no staking key.
func (fsb *FlowAPIServiceBuilder) BootstrapIdentities(addresses []string, keys []string) (flow.IdentityList, error) {
	if len(addresses) != len(keys) {
		return nil, fmt.Errorf("number of addresses and keys provided for the boostrap nodes don't match")
	}

	ids := make([]*flow.Identity, len(addresses))
	for i, address := range addresses {
		key := keys[i]

		// create the identity of the peer by setting only the relevant fields
		ids[i] = &flow.Identity{
			NodeID:        flow.ZeroID, // This is not a private network node, hence empty
			Address:       address,
			Role:          flow.RoleAccess, // the upstream node is compatible with an access node
			NetworkPubKey: nil,
		}

		// json unmarshaller needs a quotes before and after the string
		// the pflags.StringSliceVar does not retain quotes for the command line arg even if escaped with \"
		// hence this additional check to ensure the key is indeed quoted
		if !strings.HasPrefix(key, "\"") {
			key = fmt.Sprintf("\"%s\"", key)
		}

		// parse networking public key and print an error on unsecure access (=missing key)
		err := json.Unmarshal([]byte(key), &ids[i].NetworkPubKey)
		if err != nil {
			fsb.ServiceConfig.Logger.Info().Err(err)
		}
	}
	return ids, nil
}

package proxy

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/onflow/flow-go/engine/access/rpc/backend"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/utils/grpcutils"

	flowDpsAccess "github.com/GetElastech/flow-dps-access/api"
)

func NewFlowAPIService(protocolNodeAddressAndPort flow.IdentityList, executorNodeAddressAndPort flow.IdentityList, flowDpsNodeAddressAndPort flow.IdentityList, flowDpsMaxCacheSize uint64, timeout time.Duration) (*FlowAPIService, error) {
	protocolClients := make([]access.AccessAPIClient, protocolNodeAddressAndPort.Count())
	for i, identity := range protocolNodeAddressAndPort {
		identity.NetworkPubKey = nil
		if identity.NetworkPubKey == nil {
			// No public key means an insecure channel
			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithInsecure(), //nolint:staticcheck
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			protocolClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		} else {
			// Use TLS, if networking public key matches to the server
			tlsConfig, err := grpcutils.DefaultClientTLSConfig(identity.NetworkPubKey)
			if err != nil {
				return nil, err
			}

			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			protocolClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		}
	}

	executorClients := make([]access.AccessAPIClient, executorNodeAddressAndPort.Count())
	for i, identity := range executorNodeAddressAndPort {
		identity.NetworkPubKey = nil
		if identity.NetworkPubKey == nil {
			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithInsecure(), //nolint:staticcheck
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			executorClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		} else {
			tlsConfig, err := grpcutils.DefaultClientTLSConfig(identity.NetworkPubKey)
			if err != nil {
				return nil, err
			}

			clientRPCConnection, err := grpc.Dial(
				identity.Address,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
				backend.WithClientUnaryInterceptor(timeout))
			if err != nil {
				return nil, err
			}

			executorClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		}
	}

	flowDpsClients := make([]flowDpsAccess.Server, flowDpsNodeAddressAndPort.Count())
	for i, identity := range flowDpsNodeAddressAndPort {
		identity.NetworkPubKey = nil
		if identity.NetworkPubKey == nil {
			flowDpsAccessServer, err := NewDpsAccessServer(identity.Address, flowDpsMaxCacheSize, false, nil, timeout)
			if err != nil {
				return nil, err
			}

			flowDpsClients[i] = *flowDpsAccessServer

		} else {
			tlsConfig, err := grpcutils.DefaultClientTLSConfig(identity.NetworkPubKey)
			if err != nil {
				return nil, err
			}

			flowDpsAccessServer, err := NewDpsAccessServer(identity.Address, flowDpsMaxCacheSize, true, tlsConfig, timeout)
			if err != nil {
				return nil, err
			}

			flowDpsClients[i] = *flowDpsAccessServer
		}
	}

	ret := &FlowAPIService{
		upstreamFlowDps:   flowDpsClients,
		upstreamProtocol:  protocolClients,
		upstreamExecution: executorClients,
		roundRobin:        0,
		lock:              sync.Mutex{},
	}
	return ret, nil
}

type FlowAPIService struct {
	access.AccessAPIServer
	upstreamFlowDps   []flowDpsAccess.Server
	lock              sync.Mutex
	roundRobin        int
	upstreamProtocol  []access.AccessAPIClient
	upstreamExecution []access.AccessAPIClient
}

func (h *FlowAPIService) SetLocalAPI(local access.AccessAPIServer) {
	h.AccessAPIServer = local
}

func (h *FlowAPIService) clientProtocol() (access.AccessAPIClient, error) {
	if h.upstreamProtocol == nil || len(h.upstreamProtocol) == 0 {
		return nil, status.Errorf(codes.Unimplemented, "method not implemented")
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.roundRobin++
	h.roundRobin = h.roundRobin % len(h.upstreamProtocol)
	ret := h.upstreamProtocol[h.roundRobin]

	return ret, nil
}

func (h *FlowAPIService) clientExecution() (access.AccessAPIClient, error) {
	if h.upstreamExecution == nil || len(h.upstreamExecution) == 0 {
		return nil, status.Errorf(codes.Unimplemented, "method not implemented")
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.roundRobin++
	h.roundRobin = h.roundRobin % len(h.upstreamExecution)
	ret := h.upstreamExecution[h.roundRobin]

	return ret, nil
}

func (h *FlowAPIService) clientDps() (flowDpsAccess.Server, error) {
	if h.upstreamFlowDps == nil || len(h.upstreamFlowDps) == 0 {
		return flowDpsAccess.Server{}, status.Errorf(codes.Unimplemented, "method not implemented")
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.roundRobin++
	h.roundRobin = h.roundRobin % len(h.upstreamFlowDps)
	ret := h.upstreamFlowDps[h.roundRobin]

	return ret, nil
}

func (h *FlowAPIService) Ping(context context.Context, req *access.PingRequest) (*access.PingResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		return nil, err
	}
	return upstream.Ping(context, req)
}

func (h *FlowAPIService) GetLatestBlockHeader(context context.Context, req *access.GetLatestBlockHeaderRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		upstreamDps, err := h.clientDps()
		if err != nil {
			return nil, err
		}
		ret, err := upstreamDps.GetLatestBlockHeader(context, req)
		if err == nil {
			return ret, err
		}
	}
	return upstream.GetLatestBlockHeader(context, req)
}

func (h *FlowAPIService) GetBlockHeaderByID(context context.Context, req *access.GetBlockHeaderByIDRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		upstreamDps, err := h.clientDps()
		if err != nil {
			return nil, err
		}
		ret, err := upstreamDps.GetBlockHeaderByID(context, req)
		if err == nil {
			return ret, err
		}

	}
	return upstream.GetBlockHeaderByID(context, req)
}

func (h *FlowAPIService) GetBlockHeaderByHeight(context context.Context, req *access.GetBlockHeaderByHeightRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		upstreamDps, err := h.clientDps()
		if err != nil {
			return nil, err
		}
		ret, err := upstreamDps.GetBlockHeaderByHeight(context, req)
		if err == nil {
			return ret, err
		}

	}
	return upstream.GetBlockHeaderByHeight(context, req)
}

func (h *FlowAPIService) GetLatestBlock(context context.Context, req *access.GetLatestBlockRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		upstreamDps, err := h.clientDps()
		if err != nil {
			return nil, err
		}
		ret, err := upstreamDps.GetLatestBlock(context, req)
		if err == nil {
			return ret, err
		}

	}
	return upstream.GetLatestBlock(context, req)
}

func (h *FlowAPIService) GetBlockByID(context context.Context, req *access.GetBlockByIDRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		upstreamDps, err := h.clientDps()
		if err != nil {
			return nil, err
		}
		ret, err := upstreamDps.GetBlockByID(context, req)
		if err == nil {
			return ret, err
		}

	}
	return upstream.GetBlockByID(context, req)
}

func (h *FlowAPIService) GetBlockByHeight(context context.Context, req *access.GetBlockByHeightRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		upstreamDps, err := h.clientDps()
		if err != nil {
			return nil, err
		}
		ret, err := upstreamDps.GetBlockByHeight(context, req)
		if err == nil {
			return ret, err
		}

	}
	return upstream.GetBlockByHeight(context, req)
}

func (h *FlowAPIService) GetCollectionByID(context context.Context, req *access.GetCollectionByIDRequest) (*access.CollectionResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		upstreamDps, err := h.clientDps()
		if err != nil {
			return nil, err
		}
		ret, err := upstreamDps.GetCollectionByID(context, req)
		if err == nil {
			return ret, err
		}

	}
	return upstream.GetCollectionByID(context, req)
}

func (h *FlowAPIService) SendTransaction(context context.Context, req *access.SendTransactionRequest) (*access.SendTransactionResponse, error) {
	// This is a passthrough request
	// This is the only execution request that goes to BDS directly being read-write.
	upstream, err := h.clientProtocol()
	if err != nil {
		return nil, err
	}
	return upstream.SendTransaction(context, req)
}

func (h *FlowAPIService) GetTransaction(context context.Context, req *access.GetTransactionRequest) (*access.TransactionResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetTransaction(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransaction(context, req)
}

func (h *FlowAPIService) GetTransactionResult(context context.Context, req *access.GetTransactionRequest) (*access.TransactionResultResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetTransactionResult(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransactionResult(context, req)
}

func (h *FlowAPIService) GetTransactionResultByIndex(context context.Context, req *access.GetTransactionByIndexRequest) (*access.TransactionResultResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	_, err := h.clientDps()
	if err == nil {
		// TODO Is this valid still?
		// We have DPS configured, so error out instead of returning an inconsistent state conflicting with GetTransactionResult
		return nil, status.Errorf(codes.Unimplemented, "method not implemented")
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransactionResultByIndex(context, req)
}

func (h *FlowAPIService) GetAccount(context context.Context, req *access.GetAccountRequest) (*access.GetAccountResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetAccount(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccount(context, req)
}

func (h *FlowAPIService) GetAccountAtLatestBlock(context context.Context, req *access.GetAccountAtLatestBlockRequest) (*access.AccountResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetAccountAtLatestBlock(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccountAtLatestBlock(context, req)
}

func (h *FlowAPIService) GetAccountAtBlockHeight(context context.Context, req *access.GetAccountAtBlockHeightRequest) (*access.AccountResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetAccountAtBlockHeight(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccountAtBlockHeight(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtLatestBlock(context context.Context, req *access.ExecuteScriptAtLatestBlockRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.ExecuteScriptAtLatestBlock(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtLatestBlock(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtBlockID(context context.Context, req *access.ExecuteScriptAtBlockIDRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.ExecuteScriptAtBlockID(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtBlockID(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtBlockHeight(context context.Context, req *access.ExecuteScriptAtBlockHeightRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.ExecuteScriptAtBlockHeight(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtBlockHeight(context, req)
}

func (h *FlowAPIService) GetEventsForHeightRange(context context.Context, req *access.GetEventsForHeightRangeRequest) (*access.EventsResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetEventsForHeightRange(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetEventsForHeightRange(context, req)
}

func (h *FlowAPIService) GetEventsForBlockIDs(context context.Context, req *access.GetEventsForBlockIDsRequest) (*access.EventsResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetEventsForBlockIDs(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetEventsForBlockIDs(context, req)
}

func (h *FlowAPIService) GetExecutionResultForBlockID(context context.Context, req *access.GetExecutionResultForBlockIDRequest) (*access.ExecutionResultForBlockIDResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetExecutionResultForBlockID(context, req)
		if err == nil {
			return ret, err
		}
	}

	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetExecutionResultForBlockID(context, req)
}

func (h *FlowAPIService) GetNetworkParameters(context context.Context, req *access.GetNetworkParametersRequest) (*access.GetNetworkParametersResponse, error) {
	// This is a passthrough request
	// We default to DPS by default being read-only. If it fails or not set, we fall back to an execution API provider.
	// DPS may have a certain delay propagating BDS status.
	upstreamDPS, err := h.clientDps()
	if err == nil {
		ret, err := upstreamDPS.GetNetworkParameters(context, req)
		if err == nil {
			return ret, err
		}
	}
	upstream, err := h.clientExecution()
	if err != nil {
		return nil, err
	}
	return upstream.GetNetworkParameters(context, req)
}

func (h *FlowAPIService) GetLatestProtocolStateSnapshot(context context.Context, req *access.GetLatestProtocolStateSnapshotRequest) (*access.ProtocolStateSnapshotResponse, error) {
	// This is a passthrough request
	upstream, err := h.clientProtocol()
	if err != nil {
		return nil, err
	}
	return upstream.GetLatestProtocolStateSnapshot(context, req)
}

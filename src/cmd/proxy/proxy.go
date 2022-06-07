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

	upstream "github.com/onflow/api-service/m/v2/cmd/upstream/flow-dps"
)

func NewFlowAPIService(accessNodeAddressAndPort flow.IdentityList, timeout time.Duration) (*FlowAPIService, error) {
	accessClients := make([]access.AccessAPIClient, accessNodeAddressAndPort.Count())
	for i, identity := range accessNodeAddressAndPort {
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

			accessClients[i] = access.NewAccessAPIClient(clientRPCConnection)
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

			accessClients[i] = access.NewAccessAPIClient(clientRPCConnection)
		}
	}

	ret := &FlowAPIService{
		upstream:   accessClients,
		roundRobin: 0,
		lock:       sync.Mutex{},
	}
	return ret, nil
}

type FlowAPIService struct {
	access.AccessAPIServer
	upstream.Server
	lock       sync.Mutex
	roundRobin int
	upstream   []access.AccessAPIClient
}

func (h *FlowAPIService) SetLocalAPI(local access.AccessAPIServer) {
	h.AccessAPIServer = local
}

// We choose the upstream clients to pass the requests to in a round-robin fashion
// Each selection is locked. It may not be necessary but it is safer.
// The array is initialized only at startup making it read-only thereafter.
func (h *FlowAPIService) client() (access.AccessAPIClient, error) {
	if h.upstream == nil || len(h.upstream) == 0 {
		return nil, status.Errorf(codes.Unimplemented, "method not implemented")
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.roundRobin++
	h.roundRobin = h.roundRobin % len(h.upstream)
	ret := h.upstream[h.roundRobin]

	return ret, nil
}

func (h *FlowAPIService) Ping(context context.Context, req *access.PingRequest) (*access.PingResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.Ping(context, req)
}

func (h *FlowAPIService) GetLatestBlockHeader(context context.Context, req *access.GetLatestBlockHeaderRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetLatestBlockHeader(context, req)
}

func (h *FlowAPIService) GetBlockHeaderByID(context context.Context, req *access.GetBlockHeaderByIDRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetBlockHeaderByID(context, req)
}

func (h *FlowAPIService) GetBlockHeaderByHeight(context context.Context, req *access.GetBlockHeaderByHeightRequest) (*access.BlockHeaderResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetBlockHeaderByHeight(context, req)
}

func (h *FlowAPIService) GetLatestBlock(context context.Context, req *access.GetLatestBlockRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetLatestBlock(context, req)
}

func (h *FlowAPIService) GetBlockByID(context context.Context, req *access.GetBlockByIDRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetBlockByID(context, req)
}

func (h *FlowAPIService) GetBlockByHeight(context context.Context, req *access.GetBlockByHeightRequest) (*access.BlockResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetBlockByHeight(context, req)
}

func (h *FlowAPIService) GetCollectionByID(context context.Context, req *access.GetCollectionByIDRequest) (*access.CollectionResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetCollectionByID(context, req)
}

func (h *FlowAPIService) SendTransaction(context context.Context, req *access.SendTransactionRequest) (*access.SendTransactionResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.SendTransaction(context, req)
}

func (h *FlowAPIService) GetTransaction(context context.Context, req *access.GetTransactionRequest) (*access.TransactionResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransaction(context, req)
}

func (h *FlowAPIService) GetTransactionResult(context context.Context, req *access.GetTransactionRequest) (*access.TransactionResultResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransactionResult(context, req)
}

func (h *FlowAPIService) GetTransactionResultByIndex(context context.Context, req *access.GetTransactionByIndexRequest) (*access.TransactionResultResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetTransactionResultByIndex(context, req)
}

func (h *FlowAPIService) GetAccount(context context.Context, req *access.GetAccountRequest) (*access.GetAccountResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccount(context, req)
}

func (h *FlowAPIService) GetAccountAtLatestBlock(context context.Context, req *access.GetAccountAtLatestBlockRequest) (*access.AccountResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccountAtLatestBlock(context, req)
}

func (h *FlowAPIService) GetAccountAtBlockHeight(context context.Context, req *access.GetAccountAtBlockHeightRequest) (*access.AccountResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetAccountAtBlockHeight(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtLatestBlock(context context.Context, req *access.ExecuteScriptAtLatestBlockRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtLatestBlock(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtBlockID(context context.Context, req *access.ExecuteScriptAtBlockIDRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtBlockID(context, req)
}

func (h *FlowAPIService) ExecuteScriptAtBlockHeight(context context.Context, req *access.ExecuteScriptAtBlockHeightRequest) (*access.ExecuteScriptResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.ExecuteScriptAtBlockHeight(context, req)
}

func (h *FlowAPIService) GetEventsForHeightRange(context context.Context, req *access.GetEventsForHeightRangeRequest) (*access.EventsResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetEventsForHeightRange(context, req)
}

func (h *FlowAPIService) GetEventsForBlockIDs(context context.Context, req *access.GetEventsForBlockIDsRequest) (*access.EventsResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetEventsForBlockIDs(context, req)
}

func (h *FlowAPIService) GetNetworkParameters(context context.Context, req *access.GetNetworkParametersRequest) (*access.GetNetworkParametersResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetNetworkParameters(context, req)
}

func (h *FlowAPIService) GetLatestProtocolStateSnapshot(context context.Context, req *access.GetLatestProtocolStateSnapshotRequest) (*access.ProtocolStateSnapshotResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetLatestProtocolStateSnapshot(context, req)
}

func (h *FlowAPIService) GetExecutionResultForBlockID(context context.Context, req *access.GetExecutionResultForBlockIDRequest) (*access.ExecutionResultForBlockIDResponse, error) {
	// This is a passthrough request
	upstream, err := h.client()
	if err != nil {
		return nil, err
	}
	return upstream.GetExecutionResultForBlockID(context, req)
}

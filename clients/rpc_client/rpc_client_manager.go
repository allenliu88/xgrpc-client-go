package rpc_client

import (
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_request"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_response"
)

type IRpcClientManager interface {
	Request(rpcClient *rpc.RpcClient, request rpc_request.IRequest, timeoutMills uint64) (rpc_response.IResponse, error)
	CreateRpcClient(taskId string, labels map[string]string, serverRequestHandlers map[rpc.IServerRequestHandler]func() rpc_request.IRequest) *rpc.RpcClient
	GetRpcClient(labels map[string]string, serverRequestHandlers map[rpc.IServerRequestHandler]func() rpc_request.IRequest) *rpc.RpcClient
	Close(rpcClient *rpc.RpcClient)
}

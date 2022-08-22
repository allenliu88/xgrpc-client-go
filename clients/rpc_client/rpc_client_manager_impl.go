/*
 * Copyright 1999-2020 Xgrpc Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rpc_client

import (
	"strconv"
	"time"

	"github.com/allenliu88/xgrpc-client-go/common/monitor"
	"github.com/allenliu88/xgrpc-client-go/inner/uuid"

	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_request"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_response"

	"github.com/allenliu88/xgrpc-client-go/common/constant"
	"github.com/allenliu88/xgrpc-client-go/common/http_agent"
	"github.com/allenliu88/xgrpc-client-go/common/xgrpc_server"
	"github.com/allenliu88/xgrpc-client-go/util"
)

type RpcClientManager struct {
	xgrpcServer  *xgrpc_server.XgrpcServer
	clientConfig constant.ClientConfig
	uid          string
}

func NewRpcClientManager(serverConfig []constant.ServerConfig, clientConfig constant.ClientConfig, httpAgent http_agent.IHttpAgent) (IRpcClientManager, error) {
	rpcClientManager := RpcClientManager{}
	var err error
	rpcClientManager.xgrpcServer, err = xgrpc_server.NewXgrpcServer(serverConfig, clientConfig, httpAgent, clientConfig.TimeoutMs, clientConfig.Endpoint)
	rpcClientManager.clientConfig = clientConfig

	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	rpcClientManager.uid = uid.String()
	return &rpcClientManager, err
}

func (cp *RpcClientManager) Request(rpcClient *rpc.RpcClient, request rpc_request.IRequest, timeoutMills uint64) (rpc_response.IResponse, error) {
	start := time.Now()
	cp.xgrpcServer.InjectSecurityInfo(request.GetHeaders())
	cp.injectCommHeader(request.GetHeaders())
	cp.xgrpcServer.InjectSkAk(request.GetHeaders(), cp.clientConfig)
	// TODO
	// signHeaders := xgrpc_server.GetSignHeadersFromRequest(request.(rpc_request.IConfigRequest), cp.clientConfig.SecretKey)
	// request.PutAllHeaders(signHeaders)
	// TODO Config Limiter
	response, err := rpcClient.Request(request, int64(timeoutMills))
	monitor.GetConfigRequestMonitor(constant.GRPC, request.GetRequestType(), rpc_response.GetGrpcResponseStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return response, err
}

func (cp *RpcClientManager) injectCommHeader(param map[string]string) {
	now := strconv.FormatInt(util.CurrentMillis(), 10)
	param[constant.CLIENT_APPNAME_HEADER] = cp.clientConfig.AppName
	param[constant.CLIENT_REQUEST_TS_HEADER] = now
	param[constant.CLIENT_REQUEST_TOKEN_HEADER] = util.Md5(now + cp.clientConfig.AppKey)
	param[constant.EX_CONFIG_INFO] = "true"
	param[constant.CHARSET_KEY] = "utf-8"
}

func (cp *RpcClientManager) CreateRpcClient(taskId string, labels map[string]string, serverRequestHandlers map[rpc.IServerRequestHandler]func() rpc_request.IRequest) *rpc.RpcClient {
	targetLabels := map[string]string{
		constant.LABEL_SOURCE: constant.LABEL_SOURCE_SDK,
		constant.LABEL_MODULE: constant.LABEL_MODULE_CONFIG,
		"taskId":              taskId,
	}

	for k, v := range labels {
		targetLabels[k] = v
	}

	iRpcClient, _ := rpc.CreateClient(cp.uid+"-"+taskId, rpc.GRPC, targetLabels, cp.xgrpcServer)
	rpcClient := iRpcClient.GetRpcClient()
	if !rpcClient.IsInitialized() {
		// 如果不是等待初始化状态，则直接返回已有Client复用
		return rpcClient
	}

	// 注册服务器端请求处理器
	for k, v := range serverRequestHandlers {
		rpcClient.RegisterServerRequestHandler(v, k)
	}

	rpcClient.Tenant = cp.clientConfig.NamespaceId
	rpcClient.Start()

	return rpcClient
}

func (cp *RpcClientManager) GetRpcClient(labels map[string]string, serverRequestHandlers map[rpc.IServerRequestHandler]func() rpc_request.IRequest) *rpc.RpcClient {
	return cp.CreateRpcClient("0", labels, serverRequestHandlers)
}

func (cp *RpcClientManager) Close(rpcClient *rpc.RpcClient) {
	rpcClient.Shutdown()
}

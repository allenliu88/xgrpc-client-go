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

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_request"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_response"

	"github.com/allenliu88/xgrpc-client-go/util"

	xgrpc_grpc_service "github.com/allenliu88/xgrpc-client-go/api/grpc"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/grpc"
)

type GrpcConnection struct {
	*Connection
	client         xgrpc_grpc_service.RequestClient
	biStreamClient xgrpc_grpc_service.BiRequestStream_RequestBiStreamClient
}

func NewGrpcConnection(serverInfo ServerInfo, connectionId string, conn *grpc.ClientConn,
	client xgrpc_grpc_service.RequestClient, biStreamClient xgrpc_grpc_service.BiRequestStream_RequestBiStreamClient) *GrpcConnection {
	return &GrpcConnection{
		Connection: &Connection{
			serverInfo:   serverInfo,
			connectionId: connectionId,
			abandon:      false,
			conn:         conn,
		},
		client:         client,
		biStreamClient: biStreamClient,
	}
}
func (g *GrpcConnection) request(request rpc_request.IRequest, timeoutMills int64, client *RpcClient) (rpc_response.IResponse, error) {
	p := convertRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMills)*time.Millisecond)
	defer cancel()
	responsePayload, err := g.client.Request(ctx, p)
	if err != nil {
		return nil, err
	}

	responseFunc, ok := rpc_response.ClientResponseMapping[responsePayload.Metadata.GetType()]

	if !ok {
		return nil, errors.New(fmt.Sprintf("request:%s,unsupported response type:%s", request.GetRequestType(),
			responsePayload.Metadata.GetType()))
	}
	response := responseFunc()
	err = json.Unmarshal(responsePayload.GetBody().Value, response)
	return response, err
}

func (g *GrpcConnection) close() {
	g.Connection.close()
}

func (g *GrpcConnection) biStreamSend(payload *xgrpc_grpc_service.Payload) error {
	return g.biStreamClient.Send(payload)
}

func convertRequest(r rpc_request.IRequest) *xgrpc_grpc_service.Payload {
	Metadata := xgrpc_grpc_service.Metadata{
		Type:     r.GetRequestType(),
		Headers:  r.GetHeaders(),
		ClientIp: util.LocalIP(),
	}
	return &xgrpc_grpc_service.Payload{
		Metadata: &Metadata,
		Body:     &any.Any{Value: []byte(r.GetBody(r))},
	}
}

func convertResponse(r rpc_response.IResponse) *xgrpc_grpc_service.Payload {
	Metadata := xgrpc_grpc_service.Metadata{
		Type:     r.GetResponseType(),
		ClientIp: util.LocalIP(),
	}
	fmt.Println()
	fmt.Println("the response body is : " + r.GetBody())
	return &xgrpc_grpc_service.Payload{
		Metadata: &Metadata,
		Body:     &any.Any{Value: []byte(r.GetBody())},
	}
}

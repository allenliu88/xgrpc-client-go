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

package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/allenliu88/xgrpc-client-go/clients"
	"github.com/allenliu88/xgrpc-client-go/common/constant"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_request"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_response"
	"github.com/allenliu88/xgrpc-client-go/example/dto"
	"github.com/allenliu88/xgrpc-client-go/vo"
)

func main() {
	c := make(chan os.Signal)
	signal.Notify(c)

	//create ServerConfig
	sc := []constant.ServerConfig{
		*constant.NewServerConfig("127.0.0.1", 8848, constant.WithContextPath("/xgrpc"), constant.WithGrpcPort(9848)),
	}

	//create ClientConfig
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/xgrpc/log"),
		constant.WithCacheDir("/tmp/xgrpc/cache"),
		constant.WithLogLevel("debug"),
	)

	// create rpc client manager
	rpcClientManager, err := clients.NewRpcClientManager(
		vo.XgrpcClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)

	if err != nil {
		panic(err)
	}

	labels := map[string]string{"uuidName": "NameGeneratorService"}
	serverRequestHandlers := map[rpc.IServerRequestHandler]func() rpc_request.IRequest{&dto.DemoServerRequestHandler{}: func() rpc_request.IRequest {
		return dto.NewDemoServerRequest("hellWorld")
	}}

	rpcClient := rpcClientManager.GetRpcClient(labels, serverRequestHandlers)

	time.Sleep(1 * time.Second)

	iResponse, err := rpcClientManager.Request(rpcClient, dto.NewDemoRequest(), 10000)
	if err != nil {
		panic(err)
	}

	response, ok := iResponse.(*rpc_response.DemoResponse)
	if !ok {
		fmt.Errorf("DemoResponse returns type error")
	}

	if response.IsSuccess() {
		fmt.Println("======reponse msg: " + response.GetMsg())
	}

	<-c
	fmt.Println("bye!")
}

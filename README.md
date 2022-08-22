# xgrpc-client-go

Build an excellent grpc-java framework, inspired by [Nacos](https://nacos.io/zh-cn/index.html).

this repo is the go client for [xgrpc-java](https://github.com/allenliu88/xgrpc-java).

## Usage

- The server code look [here](https://github.com/allenliu88/xgrpc-java-example/tree/main/animal-name-service), the example look [here](https://github.com/allenliu88/xgrpc-java)
- The client code look [here](./example/main.go), all the following is all of about the go client usage.


## From client to server

### [Request](./example/dto/dto.go)

```go
// Client Request
type DemoRequest struct {
	*rpc_request.Request
	Module string `json:"module"`
}

func (r *DemoRequest) GetRequestType() string {
	return "DemoRequest"
}

func NewDemoRequest() *DemoRequest {
	request := rpc_request.Request{
		Headers: make(map[string]string, 8),
	}

	return &DemoRequest{
		Request: &request,
		Module:  "demo",
	}
}
```

### [Response](./common/remote/rpc/rpc_response/demo_response.go)

```go
package rpc_response

// Client Response
type DemoResponse struct {
	*Response
	Msg string `json:"msg"`
}

func (r *DemoResponse) GetMsg() string {
	return r.Msg
}

func (r *DemoResponse) GetResponseType() string {
	return "DemoResponse"
}
```

Then [register the response](./common/remote/rpc/rpc_response/rpc_response.go):

registerClientResponses:
```go
//register DemoResponse
	registerClientResponse(func() IResponse {
		return &DemoResponse{Response: &Response{}}
	})
```

### [Init and Request](./example/main.go)

```go
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
```

## From server to client

### [Server Request](./example/dto/dto.go)

```go
// Server Request
type DemoServerRequest struct {
	*rpc_request.Request
	Name   string `json:"name"`
	Module string `json:"module"`
}

func (r *DemoServerRequest) GetName() string {
	return r.Name
}

func (r *DemoServerRequest) GetRequestType() string {
	return "DemoServerRequest"
}

func NewDemoServerRequest(name string) *DemoServerRequest {
	request := rpc_request.Request{
		Headers: make(map[string]string, 8),
	}

	return &DemoServerRequest{
		Request: &request,
		Name:    name,
		Module:  "demo",
	}
}
```

### [Server Response](./example/dto/dto.go)

Note:

1. Onyl when the `ResultCode` is `constant.RESPONSE_CODE_SUCCESS`, that's `200`, the server can receive the normal response, otherwise the response in server will be `null`.
2. Must be rewrite the `GetBody()` method, otherwise, the custom field like `Msg`, will not be marshalled, and the server cannot get those fields.

```go
// Server Response
type DemoServerResponse struct {
	*rpc_response.Response
	Msg string `json:"msg"`
}

func (r *DemoServerResponse) GetMsg() string {
	return r.Msg
}

func (r *DemoServerResponse) GetBody() string {
	return util.ToJsonString(r)
}

func (r *DemoServerResponse) GetResponseType() string {
	return "DemoServerResponse"
}

func NewDemoServerResponse(msg string) *DemoServerResponse {
	return &DemoServerResponse{
		Response: &rpc_response.Response{ResultCode: constant.RESPONSE_CODE_SUCCESS}, // &rpc_response.Response{ResultCode: constant.RESPONSE_CODE_SUCCESS},
		Msg:      msg,
	}
}
```

### [Server Request Handler](./example/dto/dto.go)

```go
// Server Request Handler
type DemoServerRequestHandler struct {
}

func (c *DemoServerRequestHandler) Name() string {
	return "DemoServerRequestHandler"
}

func (c *DemoServerRequestHandler) RequestReply(request rpc_request.IRequest, rpcClient *rpc.RpcClient) rpc_response.IResponse {
	demoServerRequest, ok := request.(*DemoServerRequest)
	if ok {
		fmt.Printf("[server-push] demo server request. name=%s", demoServerRequest.Name)
	}
	return NewDemoServerResponse("hello, i'm client NameGeneratorService.")
}
```

### [The Final Step](./example/main.go)

```go
	labels := map[string]string{"uuidName": "NameGeneratorService"}
	serverRequestHandlers := map[rpc.IServerRequestHandler]func() rpc_request.IRequest{&dto.DemoServerRequestHandler{}: func() rpc_request.IRequest {
		return dto.NewDemoServerRequest("hellWorld")
	}}
```

## Output

```shell
curl -v http://127.0.0.1:9000/api/v1/animals/push
```

### Client

```shell

2022/08/23 00:29:06 [INFO] logDir:</tmp/xgrpc/log>   cacheDir:</tmp/xgrpc/cache>
======reponse msg: hello world.
[server-push] demo server request. name=AnimalNameService
the response body is : {"resultCode":200,"errorCode":0,"success":false,"message":"","requestId":"1","msg":"hello, i'm client NameGeneratorService."}
```

### Server

```shell
===========================================
HttpHeaders: [host:"127.0.0.1:9000", user-agent:"curl/7.79.1", accept:"*/*"]
===========================================
========From client connection id [1661185746534_127.0.0.1_63492], msg: hello, i'm client NameGeneratorService.
```
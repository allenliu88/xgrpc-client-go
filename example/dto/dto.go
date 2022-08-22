package dto

import (
	"fmt"

	"github.com/allenliu88/xgrpc-client-go/common/constant"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_request"
	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_response"
	"github.com/allenliu88/xgrpc-client-go/util"
)

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

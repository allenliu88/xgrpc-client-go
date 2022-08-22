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

package clients

import (
	"github.com/allenliu88/xgrpc-client-go/clients/rpc_client"
	"github.com/allenliu88/xgrpc-client-go/clients/xgrpc_client"
	"github.com/allenliu88/xgrpc-client-go/common/constant"
	"github.com/allenliu88/xgrpc-client-go/common/http_agent"
	"github.com/allenliu88/xgrpc-client-go/common/logger"
	"github.com/allenliu88/xgrpc-client-go/vo"
	"github.com/pkg/errors"
)

// CreateRpcClientManager use to create IRpcClientManager
func CreateRpcClientManager(properties map[string]interface{}) (rpcClientManager rpc_client.IRpcClientManager, err error) {
	param := getConfigParam(properties)
	return NewRpcClientManager(param)
}

func NewRpcClientManager(param vo.XgrpcClientParam) (rpcClientManager rpc_client.IRpcClientManager, err error) {
	xgrpcClient, err := setConfig(param)
	if err != nil {
		return
	}

	clientConfig, err := xgrpcClient.GetClientConfig()
	if err != nil {
		return nil, err
	}
	serverConfig, err := xgrpcClient.GetServerConfig()
	if err != nil {
		return nil, err
	}
	httpAgent, err := xgrpcClient.GetHttpAgent()
	if err != nil {
		return nil, err
	}

	if err = initLogger(clientConfig); err != nil {
		return nil, err
	}

	return rpc_client.NewRpcClientManager(serverConfig, clientConfig, httpAgent)
}

func getConfigParam(properties map[string]interface{}) (param vo.XgrpcClientParam) {

	if clientConfigTmp, exist := properties[constant.KEY_CLIENT_CONFIG]; exist {
		if clientConfig, ok := clientConfigTmp.(constant.ClientConfig); ok {
			param.ClientConfig = &clientConfig
		}
	}
	if serverConfigTmp, exist := properties[constant.KEY_SERVER_CONFIGS]; exist {
		if serverConfigs, ok := serverConfigTmp.([]constant.ServerConfig); ok {
			param.ServerConfigs = serverConfigs
		}
	}
	return
}

func setConfig(param vo.XgrpcClientParam) (iClient xgrpc_client.IXgrpcClient, err error) {
	client := &xgrpc_client.XgrpcClient{}
	if param.ClientConfig == nil {
		// default clientConfig
		_ = client.SetClientConfig(constant.ClientConfig{
			TimeoutMs:    10 * 1000,
			BeatInterval: 5 * 1000,
		})
	} else {
		err = client.SetClientConfig(*param.ClientConfig)
		if err != nil {
			return nil, err
		}
	}

	if len(param.ServerConfigs) == 0 {
		clientConfig, _ := client.GetClientConfig()
		if len(clientConfig.Endpoint) <= 0 {
			err = errors.New("server configs not found in properties")
			return nil, err
		}
		_ = client.SetServerConfig(nil)
	} else {
		err = client.SetServerConfig(param.ServerConfigs)
		if err != nil {
			return nil, err
		}
	}

	if _, _err := client.GetHttpAgent(); _err != nil {
		if clientCfg, err := client.GetClientConfig(); err == nil {
			_ = client.SetHttpAgent(&http_agent.HttpAgent{TlsConfig: clientCfg.TLSCfg})
		}
	}
	iClient = client
	return
}

func initLogger(clientConfig constant.ClientConfig) error {
	return logger.InitLogger(logger.BuildLoggerConfig(clientConfig))
}

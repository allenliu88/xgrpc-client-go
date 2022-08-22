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

package xgrpc_client

import (
	"log"
	"os"
	"strconv"

	"github.com/pkg/errors"

	"github.com/allenliu88/xgrpc-client-go/common/constant"
	"github.com/allenliu88/xgrpc-client-go/common/file"
	"github.com/allenliu88/xgrpc-client-go/common/http_agent"
)

type XgrpcClient struct {
	clientConfigValid  bool
	serverConfigsValid bool
	agent              http_agent.IHttpAgent
	clientConfig       constant.ClientConfig
	serverConfigs      []constant.ServerConfig
}

//SetClientConfig is use to set xgrpc client Config
func (client *XgrpcClient) SetClientConfig(config constant.ClientConfig) (err error) {
	if config.TimeoutMs <= 0 {
		config.TimeoutMs = 10 * 1000
	}

	if config.BeatInterval <= 0 {
		config.BeatInterval = 5 * 1000
	}

	if config.UpdateThreadNum <= 0 {
		config.UpdateThreadNum = 20
	}

	if len(config.LogLevel) == 0 {
		config.LogLevel = "info"
	}

	if config.CacheDir == "" {
		config.CacheDir = file.GetCurrentPath() + string(os.PathSeparator) + "cache"
	}

	if config.LogDir == "" {
		config.LogDir = file.GetCurrentPath() + string(os.PathSeparator) + "log"
	}
	log.Printf("[INFO] logDir:<%s>   cacheDir:<%s>", config.LogDir, config.CacheDir)
	client.clientConfig = config
	client.clientConfigValid = true

	return
}

//SetServerConfig is use to set xgrpc server config
func (client *XgrpcClient) SetServerConfig(configs []constant.ServerConfig) (err error) {
	if len(configs) <= 0 {
		//it's may be use endpoint to get xgrpc server address
		client.serverConfigsValid = true
		return
	}

	for i := 0; i < len(configs); i++ {
		if len(configs[i].IpAddr) <= 0 || configs[i].Port <= 0 || configs[i].Port > 65535 {
			err = errors.New("[client.SetServerConfig] configs[" + strconv.Itoa(i) + "] is invalid")
			return
		}
		if len(configs[i].ContextPath) <= 0 {
			configs[i].ContextPath = constant.DEFAULT_CONTEXT_PATH
		}
		if len(configs[i].Scheme) <= 0 {
			configs[i].Scheme = constant.DEFAULT_SERVER_SCHEME
		}
	}
	client.serverConfigs = configs
	client.serverConfigsValid = true
	return
}

//GetClientConfig use to get client config
func (client *XgrpcClient) GetClientConfig() (config constant.ClientConfig, err error) {
	config = client.clientConfig
	if !client.clientConfigValid {
		err = errors.New("[client.GetClientConfig] invalid client config")
	}
	return
}

//GetServerConfig use to get server config
func (client *XgrpcClient) GetServerConfig() (configs []constant.ServerConfig, err error) {
	configs = client.serverConfigs
	if !client.serverConfigsValid {
		err = errors.New("[client.GetServerConfig] invalid server configs")
	}
	return
}

//SetHttpAgent use to set http agent
func (client *XgrpcClient) SetHttpAgent(agent http_agent.IHttpAgent) (err error) {
	if agent == nil {
		err = errors.New("[client.SetHttpAgent] http agent can not be nil")
	} else {
		client.agent = agent
	}
	return
}

//GetHttpAgent use to get http agent
func (client *XgrpcClient) GetHttpAgent() (agent http_agent.IHttpAgent, err error) {
	if client.agent == nil {
		err = errors.New("[client.GetHttpAgent] invalid http agent")
	} else {
		agent = client.agent
	}
	return
}

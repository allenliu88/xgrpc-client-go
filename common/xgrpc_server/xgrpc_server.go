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

package xgrpc_server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/allenliu88/xgrpc-client-go/common/remote/rpc/rpc_request"

	"github.com/allenliu88/xgrpc-client-go/common/monitor"

	"github.com/allenliu88/xgrpc-client-go/common/constant"
	"github.com/allenliu88/xgrpc-client-go/common/http_agent"
	"github.com/allenliu88/xgrpc-client-go/common/logger"
	"github.com/allenliu88/xgrpc-client-go/common/security"
	"github.com/allenliu88/xgrpc-client-go/common/xgrpc_error"
	"github.com/allenliu88/xgrpc-client-go/inner/uuid"
	"github.com/allenliu88/xgrpc-client-go/util"
)

type XgrpcServer struct {
	sync.RWMutex
	securityLogin         security.AuthClient
	serverList            []constant.ServerConfig
	httpAgent             http_agent.IHttpAgent
	timeoutMs             uint64
	endpoint              string
	lastSrvRefTime        int64
	vipSrvRefInterMills   int64
	contextPath           string
	currentIndex          int32
	ServerSrcChangeSignal chan struct{}
}

func NewXgrpcServer(serverList []constant.ServerConfig, clientCfg constant.ClientConfig, httpAgent http_agent.IHttpAgent, timeoutMs uint64, endpoint string) (*XgrpcServer, error) {
	severLen := len(serverList)
	if severLen == 0 && endpoint == "" {
		return &XgrpcServer{}, errors.New("both serverlist  and  endpoint are empty")
	}

	securityLogin := security.NewAuthClient(clientCfg, serverList, httpAgent)

	ns := XgrpcServer{
		serverList:            serverList,
		securityLogin:         securityLogin,
		httpAgent:             httpAgent,
		timeoutMs:             timeoutMs,
		endpoint:              endpoint,
		vipSrvRefInterMills:   10000,
		contextPath:           clientCfg.ContextPath,
		ServerSrcChangeSignal: make(chan struct{}, 1),
	}
	if severLen > 0 {
		ns.currentIndex = rand.Int31n(int32(severLen))
	}

	ns.initRefreshSrvIfNeed()
	_, err := securityLogin.Login()

	if err != nil {
		return &ns, err
	}

	securityLogin.AutoRefresh()
	return &ns, nil
}

func (server *XgrpcServer) callConfigServer(api string, params map[string]string, newHeaders map[string]string,
	method string, curServer string, contextPath string, timeoutMS uint64) (result string, err error) {
	start := time.Now()
	if contextPath == "" {
		contextPath = constant.WEB_CONTEXT
	}

	signHeaders := GetSignHeaders(params, newHeaders["secretKey"])

	url := curServer + contextPath + api

	headers := map[string][]string{}
	for k, v := range newHeaders {
		if k != "accessKey" && k != "secretKey" {
			headers[k] = []string{v}
		}
	}
	headers["Client-Version"] = []string{constant.CLIENT_VERSION}
	headers["User-Agent"] = []string{constant.CLIENT_VERSION}
	//headers["Accept-Encoding"] = []string{"gzip,deflate,sdch"}
	headers["Connection"] = []string{"Keep-Alive"}
	headers["exConfigInfo"] = []string{"true"}
	uid, err := uuid.NewV4()
	if err != nil {
		return
	}
	headers["RequestId"] = []string{uid.String()}
	headers["Content-Type"] = []string{"application/x-www-form-urlencoded;charset=utf-8"}
	headers["Spas-AccessKey"] = []string{newHeaders["accessKey"]}
	headers["Timestamp"] = []string{signHeaders["Timestamp"]}
	headers["Spas-Signature"] = []string{signHeaders["Spas-Signature"]}
	server.InjectSecurityInfo(params)

	var response *http.Response
	response, err = server.httpAgent.Request(method, url, headers, timeoutMS, params)
	monitor.GetConfigRequestMonitor(method, url, util.GetStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	if err != nil {
		return
	}
	var bytes []byte
	bytes, err = ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return
	}
	result = string(bytes)
	if response.StatusCode == constant.RESPONSE_CODE_SUCCESS {
		return
	} else {
		err = xgrpc_error.NewXgrpcError(strconv.Itoa(response.StatusCode), string(bytes), nil)
		return
	}
}

func (server *XgrpcServer) callServer(api string, params map[string]string, method string, curServer string, contextPath string) (result string, err error) {
	start := time.Now()
	if contextPath == "" {
		contextPath = constant.WEB_CONTEXT
	}

	url := curServer + contextPath + api

	headers := map[string][]string{}
	headers["Client-Version"] = []string{constant.CLIENT_VERSION}
	headers["User-Agent"] = []string{constant.CLIENT_VERSION}
	//headers["Accept-Encoding"] = []string{"gzip,deflate,sdch"}
	headers["Connection"] = []string{"Keep-Alive"}
	uid, err := uuid.NewV4()
	if err != nil {
		return
	}
	headers["RequestId"] = []string{uid.String()}
	headers["Request-Module"] = []string{"Naming"}
	headers["Content-Type"] = []string{"application/x-www-form-urlencoded;charset=utf-8"}

	server.InjectSecurityInfo(params)

	var response *http.Response
	response, err = server.httpAgent.Request(method, url, headers, server.timeoutMs, params)
	if err != nil {
		return
	}
	var bytes []byte
	bytes, err = ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return
	}
	result = string(bytes)
	monitor.GetNamingRequestMonitor(method, api, util.GetStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	if response.StatusCode == constant.RESPONSE_CODE_SUCCESS {
		return
	} else {
		err = errors.New(fmt.Sprintf("request return error code %d", response.StatusCode))
		return
	}
}

func (server *XgrpcServer) ReqConfigApi(api string, params map[string]string, headers map[string]string, method string, timeoutMS uint64) (string, error) {
	srvs := server.serverList
	if srvs == nil || len(srvs) == 0 {
		return "", errors.New("server list is empty")
	}

	server.InjectSecurityInfo(params)

	//only one server,retry request when error
	var err error
	var result string
	if len(srvs) == 1 {
		for i := 0; i < constant.REQUEST_DOMAIN_RETRY_TIME; i++ {
			result, err = server.callConfigServer(api, params, headers, method, getAddress(srvs[0]), srvs[0].ContextPath, timeoutMS)
			if err == nil {
				return result, nil
			}
			logger.Errorf("api<%s>,method:<%s>, params:<%s>, call domain error:<%+v> , result:<%s>", api, method, util.ToJsonString(params), err, result)
		}
	} else {
		index := rand.Intn(len(srvs))
		for i := 1; i <= len(srvs); i++ {
			curServer := srvs[index]
			result, err = server.callConfigServer(api, params, headers, method, getAddress(curServer), curServer.ContextPath, timeoutMS)
			if err == nil {
				return result, nil
			}
			logger.Errorf("[ERROR] api<%s>,method:<%s>, params:<%s>, call domain error:<%+v> , result:<%s> \n", api, method, util.ToJsonString(params), err, result)
			index = (index + i) % len(srvs)
		}
	}
	return "", errors.Wrapf(err, "retry %d times request failed!", constant.REQUEST_DOMAIN_RETRY_TIME)
}

func (server *XgrpcServer) ReqApi(api string, params map[string]string, method string) (string, error) {
	srvs := server.serverList
	if srvs == nil || len(srvs) == 0 {
		return "", errors.New("server list is empty")
	}

	server.InjectSecurityInfo(params)

	//only one server,retry request when error
	var err error
	var result string
	if len(srvs) == 1 {
		for i := 0; i < constant.REQUEST_DOMAIN_RETRY_TIME; i++ {
			result, err = server.callServer(api, params, method, getAddress(srvs[0]), srvs[0].ContextPath)
			if err == nil {
				return result, nil
			}
			logger.Errorf("api<%s>,method:<%s>, params:<%s>, call domain error:<%+v> , result:<%s>", api, method, util.ToJsonString(params), err, result)
		}
	} else {
		index := rand.Intn(len(srvs))
		for i := 1; i <= len(srvs); i++ {
			curServer := srvs[index]
			result, err = server.callServer(api, params, method, getAddress(curServer), curServer.ContextPath)
			if err == nil {
				return result, nil
			}
			logger.Errorf("api<%s>,method:<%s>, params:<%s>, call domain error:<%+v> , result:<%s>", api, method, util.ToJsonString(params), err, result)
			index = (index + i) % len(srvs)
		}
	}
	return "", errors.Wrapf(err, "retry %d times request failed!", constant.REQUEST_DOMAIN_RETRY_TIME)
}

func (server *XgrpcServer) initRefreshSrvIfNeed() {
	if server.endpoint == "" {
		return
	}
	server.refreshServerSrvIfNeed()
	go func() {
		for {
			time.Sleep(time.Duration(1) * time.Second)
			server.refreshServerSrvIfNeed()
		}
	}()

}

func (server *XgrpcServer) refreshServerSrvIfNeed() {
	if util.CurrentMillis()-server.lastSrvRefTime < server.vipSrvRefInterMills && len(server.serverList) > 0 {
		return
	}

	var list []string
	urlString := "http://" + server.endpoint + "/xgrpc/serverlist"
	result := server.httpAgent.RequestOnlyResult(http.MethodGet, urlString, nil, server.timeoutMs, nil)
	list = strings.Split(result, "\n")
	logger.Infof("http xgrpc server list: <%s>", result)

	var servers []constant.ServerConfig
	contextPath := server.contextPath
	if len(contextPath) == 0 {
		contextPath = constant.WEB_CONTEXT
	}
	for _, line := range list {
		if line != "" {
			splitLine := strings.Split(strings.TrimSpace(line), ":")
			port := 8848
			var err error
			if len(splitLine) == 2 {
				port, err = strconv.Atoi(splitLine[1])
				if err != nil {
					logger.Errorf("get port from server:<%s>  error: <%+v>", line, err)
					continue
				}
			}

			servers = append(servers, constant.ServerConfig{Scheme: constant.DEFAULT_SERVER_SCHEME, IpAddr: splitLine[0], Port: uint64(port), ContextPath: contextPath})
		}
	}
	if len(servers) > 0 {
		if !reflect.DeepEqual(server.serverList, servers) {
			server.Lock()
			logger.Infof("server list is updated, old: <%v>,new:<%v>", server.serverList, servers)
			server.serverList = servers
			server.ServerSrcChangeSignal <- struct{}{}
			server.lastSrvRefTime = util.CurrentMillis()
			server.Unlock()
		}

	}
	return
}

func (server *XgrpcServer) GetServerList() []constant.ServerConfig {
	return server.serverList
}

func (server *XgrpcServer) InjectSecurityInfo(param map[string]string) {
	accessToken := server.securityLogin.GetAccessToken()
	if accessToken != "" {
		param[constant.KEY_ACCESS_TOKEN] = accessToken
	}
}

func (server *XgrpcServer) InjectSign(request rpc_request.IRequest, param map[string]string, clientConfig constant.ClientConfig) {
	if clientConfig.AccessKey == "" || clientConfig.SecretKey == "" {
		return
	}
	sts := request.GetStringToSign()
	if sts == "" {
		return
	}
	signature := signWithhmacSHA1Encrypt(sts, clientConfig.SecretKey)
	param["data"] = sts
	param["signature"] = signature
	param["ak"] = clientConfig.AccessKey
}

func getAddress(cfg constant.ServerConfig) string {
	if strings.Index(cfg.IpAddr, "http://") >= 0 || strings.Index(cfg.IpAddr, "https://") >= 0 {
		return cfg.IpAddr + ":" + strconv.Itoa(int(cfg.Port))
	}
	return cfg.Scheme + "://" + cfg.IpAddr + ":" + strconv.Itoa(int(cfg.Port))
}

func GetSignHeadersFromRequest(cr rpc_request.IConfigRequest, secretKey string) map[string]string {
	resource := ""

	if len(cr.GetGroup()) != 0 {
		resource = cr.GetTenant() + "+" + cr.GetGroup()
	} else {
		resource = cr.GetGroup()
	}

	headers := map[string]string{}

	timeStamp := strconv.FormatInt(util.CurrentMillis(), 10)
	headers["Timestamp"] = timeStamp

	signature := ""

	if resource == "" {
		signature = signWithhmacSHA1Encrypt(timeStamp, secretKey)
	} else {
		signature = signWithhmacSHA1Encrypt(resource+"+"+timeStamp, secretKey)
	}

	headers["Spas-Signature"] = signature

	return headers
}

func GetSignHeaders(params map[string]string, secretKey string) map[string]string {
	resource := ""

	if len(params["tenant"]) != 0 {
		resource = params["tenant"] + "+" + params["group"]
	} else {
		resource = params["group"]
	}

	headers := map[string]string{}

	timeStamp := strconv.FormatInt(util.CurrentMillis(), 10)
	headers["Timestamp"] = timeStamp

	signature := ""

	if resource == "" {
		signature = signWithhmacSHA1Encrypt(timeStamp, secretKey)
	} else {
		signature = signWithhmacSHA1Encrypt(resource+"+"+timeStamp, secretKey)
	}

	headers["Spas-Signature"] = signature

	return headers
}

func signWithhmacSHA1Encrypt(encryptText, encryptKey string) string {
	//hmac ,use sha1
	key := []byte(encryptKey)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(encryptText))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (server *XgrpcServer) GetNextServer() (constant.ServerConfig, error) {
	serverLen := len(server.GetServerList())
	if serverLen == 0 {
		return constant.ServerConfig{}, errors.New("server is empty")
	}
	index := atomic.AddInt32(&server.currentIndex, 1) % int32(serverLen)
	return server.GetServerList()[index], nil
}

func (server *XgrpcServer) InjectSkAk(params map[string]string, clientConfig constant.ClientConfig) {
	if clientConfig.AccessKey != "" {
		params["Spas-AccessKey"] = clientConfig.AccessKey
	}
}

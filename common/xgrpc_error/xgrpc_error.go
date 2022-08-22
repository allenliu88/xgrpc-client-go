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

package xgrpc_error

import (
	"fmt"

	"github.com/allenliu88/xgrpc-client-go/common/constant"
)

type XgrpcError struct {
	errorCode   string
	errMsg      string
	originError error
}

func NewXgrpcError(errorCode string, errMsg string, originError error) *XgrpcError {
	return &XgrpcError{
		errorCode:   errorCode,
		errMsg:      errMsg,
		originError: originError,
	}

}

func (err *XgrpcError) Error() (str string) {
	xgrpcErrMsg := fmt.Sprintf("[%s] %s", err.ErrorCode(), err.errMsg)
	if err.originError != nil {
		return xgrpcErrMsg + "\ncaused by:\n" + err.originError.Error()
	}
	return xgrpcErrMsg
}

func (err *XgrpcError) ErrorCode() string {
	if err.errorCode == "" {
		return constant.DefaultClientErrorCode
	} else {
		return err.errorCode
	}
}

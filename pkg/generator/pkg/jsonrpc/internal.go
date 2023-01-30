package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (client *ClientRPC) newRequest(ctx context.Context, reqBody interface{}) (request *http.Request, err error) {

	var body []byte
	if body, err = json.Marshal(reqBody); err != nil {
		return
	}
	if request, err = http.NewRequestWithContext(ctx, "POST", client.endpoint, bytes.NewReader(body)); err != nil {
		return
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	for k, v := range client.options.customHeaders {
		if k == "Host" {
			request.Host = v
		} else {
			request.Header.Set(k, v)
		}
	}
	for _, header := range client.options.headersFromCtx {
		if value := ctx.Value(header); value != nil {
			if k := toString(header); k != "" {
				if v := toString(value); v != "" {
					request.Header.Set(k, v)
				}
			}
		}
	}
	return
}

func (client *ClientRPC) doCall(ctx context.Context, request *RequestRPC) (rpcResponse *ResponseRPC, err error) {

	var httpRequest *http.Request
	if httpRequest, err = client.newRequest(ctx, request); err != nil {
		err = fmt.Errorf("rpc call %v() on %v: %v", request.Method, client.endpoint, err.Error())
		return
	}
	if client.options.logRequests {
		if cmd, cmdErr := toCurl(httpRequest); cmdErr == nil {
			log.Ctx(ctx).Debug().Str("method", request.Method).Str("curl", cmd.String()).Msg("call")
		}
	}
	defer func() {
		if err != nil && client.options.logOnError {
			if cmd, cmdErr := toCurl(httpRequest); cmdErr == nil {
				log.Ctx(ctx).Error().Str("method", request.Method).Str("curl", cmd.String()).Msg("call")
			}
		}
	}()
	var httpResponse *http.Response
	if httpResponse, err = client.httpClient.Do(httpRequest); err != nil {
		err = fmt.Errorf("rpc call %v() on %v: %v", request.Method, httpRequest.URL.String(), err.Error())
		return
	}
	defer httpResponse.Body.Close()
	decoder := json.NewDecoder(httpResponse.Body)
	if !client.options.allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	decoder.UseNumber()
	err = decoder.Decode(&rpcResponse)
	if err != nil {
		if httpResponse.StatusCode >= 400 {
			return nil, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %v", request.Method, httpRequest.URL.String(), httpResponse.StatusCode, err.Error()),
			}
		}
		return nil, fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %v", request.Method, httpRequest.URL.String(), httpResponse.StatusCode, err.Error())
	}
	if rpcResponse == nil {
		if httpResponse.StatusCode >= 400 {
			return nil, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", request.Method, httpRequest.URL.String(), httpResponse.StatusCode),
			}
		}
		err = fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", request.Method, httpRequest.URL.String(), httpResponse.StatusCode)
		return
	}
	if httpResponse.StatusCode >= 400 {
		if rpcResponse.Error != nil {
			return rpcResponse, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc call %v() on %v status code: %v. rpc response error: %v", request.Method, httpRequest.URL.String(), httpResponse.StatusCode, rpcResponse.Error),
			}
		}
		return rpcResponse, &HTTPError{
			Code: httpResponse.StatusCode,
			err:  fmt.Errorf("rpc call %v() on %v status code: %v. no rpc error available", request.Method, httpRequest.URL.String(), httpResponse.StatusCode),
		}
	}
	return
}

func (client *ClientRPC) doBatchCall(ctx context.Context, rpcRequests []*RequestRPC) (rpcResponses ResponsesRPC, err error) {

	defer func() {
		if err != nil {
			for _, request := range rpcRequests {
				if request.ID == NilID {
					continue
				}
				rpcResponses = append(rpcResponses, &ResponseRPC{
					ID:      request.ID,
					JSONRPC: request.JSONRPC,
					Error: &RPCError{
						Message: err.Error(),
					},
				})
			}
		}
	}()
	var httpRequest *http.Request
	if httpRequest, err = client.newRequest(ctx, rpcRequests); err != nil {
		err = fmt.Errorf("rpc batch call on %v: %v", client.endpoint, err.Error())
		return
	}
	if client.options.logRequests {
		if cmd, cmdErr := toCurl(httpRequest); cmdErr == nil {
			log.Ctx(ctx).Debug().Str("method", "batch").Int("count", len(rpcRequests)).Str("curl", cmd.String()).Msg("call")
		}
	}
	defer func() {
		if err != nil && client.options.logOnError {
			if cmd, cmdErr := toCurl(httpRequest); cmdErr == nil {
				log.Ctx(ctx).Error().Str("method", "batch").Int("count", len(rpcRequests)).Str("curl", cmd.String()).Msg("call")
			}
		}
	}()
	var httpResponse *http.Response
	if httpResponse, err = client.httpClient.Do(httpRequest); err != nil {
		err = fmt.Errorf("rpc batch call on %v: %v", httpRequest.URL.String(), err.Error())
		return
	}
	defer httpResponse.Body.Close()
	decoder := json.NewDecoder(httpResponse.Body)
	if !client.options.allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	decoder.UseNumber()
	err = decoder.Decode(&rpcResponses)
	if err != nil {
		if httpResponse.StatusCode >= 400 {
			return nil, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc batch call on %v status code: %v. could not decode body to rpc response: %v", httpRequest.URL.String(), httpResponse.StatusCode, err.Error()),
			}
		}
		err = fmt.Errorf("rpc batch call on %v status code: %v. could not decode body to rpc response: %v", httpRequest.URL.String(), httpResponse.StatusCode, err.Error())
		return
	}
	if len(rpcResponses) == 0 {
		if httpResponse.StatusCode >= 400 {
			return nil, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc batch call on %v status code: %v. rpc response missing", httpRequest.URL.String(), httpResponse.StatusCode),
			}
		}
		err = fmt.Errorf("rpc batch call on %v status code: %v. rpc response missing", httpRequest.URL.String(), httpResponse.StatusCode)
		return
	}
	if httpResponse.StatusCode >= 400 {
		return rpcResponses, &HTTPError{
			Code: httpResponse.StatusCode,
			err:  fmt.Errorf("rpc batch call on %v status code: %v. check rpc responses for potential rpc error", httpRequest.URL.String(), httpResponse.StatusCode),
		}
	}
	return
}

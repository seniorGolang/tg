// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// RegisterNetFunctions регистрирует все сетевые функции в модуле net.
func RegisterNetFunctions(builder wazero.HostModuleBuilder, h *host.Host, nm *netManager, hm *httpManager) {

	// Функции для работы с соединениями
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {
			return connDial(ctx, h, nm, networkPtr, networkLen, addressPtr, addressLen, connIDPtr)
		}).
		Export("conn_dial")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, deadline uint64, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {
			return connDialContext(ctx, h, nm, deadline, networkPtr, networkLen, addressPtr, addressLen, connIDPtr)
		}).
		Export("conn_dial_context")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {
			return connDialTLS(ctx, h, nm, networkPtr, networkLen, addressPtr, addressLen, connIDPtr)
		}).
		Export("conn_dial_tls")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, deadline uint64, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {
			return connDialTLSContext(ctx, h, nm, deadline, networkPtr, networkLen, addressPtr, addressLen, connIDPtr)
		}).
		Export("conn_dial_tls_context")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, configPtr uint32, configLen uint32, connIDPtr uint32) (result uint64) {
			return connDialTLSWithConfig(ctx, h, nm, networkPtr, networkLen, addressPtr, addressLen, configPtr, configLen, connIDPtr)
		}).
		Export("conn_dial_tls_with_config")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64) (result uint64) {
			return connTLSHandshake(ctx, h, nm, connID)
		}).
		Export("conn_tls_handshake")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64, bufferPtrPtr uint32) (result uint64) {
			return connGetBufferPtr(ctx, h, nm, connID, bufferPtrPtr)
		}).
		Export("conn_get_buffer_ptr")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64, bufferPtrPtr uint32) (result uint64) {
			return connGetWriteBufferPtr(ctx, h, nm, connID, bufferPtrPtr)
		}).
		Export("conn_get_write_buffer_ptr")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64) (result uint64) {
			return connClose(ctx, h, nm, connID)
		}).
		Export("conn_close")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64, addrPtr uint32, addrLenPtr uint32) (result uint64) {
			return connRemoteAddr(ctx, h, nm, connID, addrPtr, addrLenPtr)
		}).
		Export("conn_remote_addr")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64, addrPtr uint32, addrLenPtr uint32) (result uint64) {
			return connLocalAddr(ctx, h, nm, connID, addrPtr, addrLenPtr)
		}).
		Export("conn_local_addr")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64, deadline uint64) (result uint64) {
			return connSetDeadline(ctx, h, nm, connID, deadline)
		}).
		Export("conn_set_deadline")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64, deadline uint64) (result uint64) {
			return connSetReadDeadline(ctx, h, nm, connID, deadline)
		}).
		Export("conn_set_read_deadline")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, connID uint64, deadline uint64) (result uint64) {
			return connSetWriteDeadline(ctx, h, nm, connID, deadline)
		}).
		Export("conn_set_write_deadline")

	// Функции для работы со слушателями
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, listenerIDPtr uint32) (result uint64) {
			return listenerListen(ctx, h, nm, networkPtr, networkLen, addressPtr, addressLen, listenerIDPtr)
		}).
		Export("listener_listen")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, listenerID uint64, connIDPtr uint32) (result uint64) {
			return listenerAccept(ctx, h, nm, listenerID, connIDPtr)
		}).
		Export("listener_accept")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, listenerID uint64) (result uint64) {
			return listenerClose(ctx, h, nm, listenerID)
		}).
		Export("listener_close")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, listenerID uint64, addrPtr uint32, addrLenPtr uint32) (result uint64) {
			return listenerAddr(ctx, h, nm, listenerID, addrPtr, addrLenPtr)
		}).
		Export("listener_addr")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, listenerID uint64, callbackFuncPtr uint32, callbackFuncLen uint32) (result uint64) {
			return listenerServeStart(ctx, h, nm, listenerID, callbackFuncPtr, callbackFuncLen)
		}).
		Export("listener_serve_start")

	// HTTP функции
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, addrPtr uint32, addrLen uint32, handlerID uint64) (result uint64) {
			return hostListenAndServe(ctx, h, hm, addrPtr, addrLen, handlerID)
		}).
		Export("host_listen_and_serve")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, requestIDPtr uint32, handlerIDPtr uint32) (result uint64) {
			return hostGetNextRequest(ctx, h, hm, requestIDPtr, handlerIDPtr)
		}).
		Export("host_get_next_request")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, requestID uint64, infoBufPtr uint32, infoBufLen uint32) (result uint64) {
			return hostGetRequestInfo(ctx, h, hm, requestID, infoBufPtr, infoBufLen)
		}).
		Export("host_get_request_info")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, requestID uint64, bufPtr uint32, bufLen uint32) (result uint64) {
			return hostReadRequestBody(ctx, h, hm, requestID, bufPtr, bufLen)
		}).
		Export("host_read_request_body")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, requestID uint64, statusCode int32, headersPtr uint32, headersLen uint32) (result uint64) {
			return hostWriteResponseHeaders(ctx, h, hm, requestID, statusCode, headersPtr, headersLen)
		}).
		Export("host_write_response_headers")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, requestID uint64, dataPtr uint32, dataLen uint32) (result uint64) {
			return hostWriteResponseBody(ctx, h, hm, requestID, dataPtr, dataLen)
		}).
		Export("host_write_response_body")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, requestID uint64) (result uint64) {
			return hostFinishRequest(ctx, h, hm, requestID)
		}).
		Export("host_finish_request")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, serverID uint64) (result uint64) {
			return hostStopServer(ctx, h, hm, serverID)
		}).
		Export("host_stop_server")
}

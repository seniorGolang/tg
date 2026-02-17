// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/tetratelabs/wazero/sys"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

func safeIntToUint32(val int) (result uint32, err error) {

	if val < 0 || val > math.MaxUint32 {
		return 0, errors.New(i18n.Msg("value exceeds uint32 range"))
	}

	result = uint32(val)
	return
}

type RequestContext struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	DoneChan       chan struct{}
	BodyReader     *bufio.Reader
	BodyBuffer     []byte
	BodyOffset     int
	HandlerID      uint64
}

type httpServerState struct {
	Server   *http.Server
	Addr     string
	Handler  http.Handler
	DoneChan chan struct{}
}

type httpManager struct {
	mu sync.RWMutex

	serverID     uint64
	serverMap    map[uint64]*httpServerState
	handlerID    uint64
	handlerMap   map[uint64]http.Handler
	requestID    uint64
	requestMap   map[uint64]*RequestContext
	requestQueue chan uint64 // небуферизованная — блокирует до обработки

	processorStop   chan struct{}
	processorDone   chan struct{}
	processorCtx    context.Context
	processorCancel context.CancelFunc
}

func NewHTTPManager() (hm *httpManager) {
	return &httpManager{
		serverMap:     make(map[uint64]*httpServerState),
		handlerMap:    make(map[uint64]http.Handler),
		requestMap:    make(map[uint64]*RequestContext),
		requestQueue:  make(chan uint64),
		processorStop: make(chan struct{}),
		processorDone: make(chan struct{}),
	}
}

func (hm *httpManager) StoreHandler(handler http.Handler) (handlerID uint64) {

	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.handlerID++
	hm.handlerMap[hm.handlerID] = handler

	handlerID = hm.handlerID
	return
}

func (hm *httpManager) GetHandler(handlerID uint64) (handler http.Handler, err error) {

	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var ok bool
	if handler, ok = hm.handlerMap[handlerID]; !ok {
		return nil, fmt.Errorf(i18n.Msg("handler id %d does not exist"), handlerID)
	}

	return
}

func (hm *httpManager) StoreServer(server *http.Server, addr string, handler http.Handler) (serverID uint64) {

	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.serverID++
	hm.serverMap[hm.serverID] = &httpServerState{
		Server:   server,
		Addr:     addr,
		Handler:  handler, // Может быть nil, handler находится в плагине
		DoneChan: make(chan struct{}),
	}

	serverID = hm.serverID
	return
}

func (hm *httpManager) GetServer(serverID uint64) (state *httpServerState, err error) {

	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var ok bool
	if state, ok = hm.serverMap[serverID]; !ok {
		return nil, fmt.Errorf(i18n.Msg("server id %d does not exist"), serverID)
	}

	return
}

func (hm *httpManager) DelServer(serverID uint64) {

	hm.mu.Lock()
	delete(hm.serverMap, serverID)
	serverCount := len(hm.serverMap)
	hm.mu.Unlock()

	if serverCount == 0 {
		hm.StopProcessor()
	}
}

func (hm *httpManager) StoreRequest(ctx *RequestContext) (requestID uint64) {

	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.requestID++
	hm.requestMap[hm.requestID] = ctx

	requestID = hm.requestID
	return
}

func (hm *httpManager) GetRequest(requestID uint64) (ctx *RequestContext, err error) {

	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var ok bool
	if ctx, ok = hm.requestMap[requestID]; !ok {
		return nil, fmt.Errorf(i18n.Msg("request id %d does not exist"), requestID)
	}

	return
}

func (hm *httpManager) DelRequest(requestID uint64) {

	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.requestMap, requestID)
}

// EnqueueRequest: небуферизованная requestQueue блокирует до появления потребителя.
func (hm *httpManager) EnqueueRequest(requestID uint64) {

	hm.requestQueue <- requestID
}

// StartProcessor запускает цикл: процессор вызывает _dispatch, та — host_get_next_request (блокируется до запроса).
func (hm *httpManager) StartProcessor(ctx context.Context, h *host.Host) {

	hm.mu.Lock()
	if hm.processorCancel != nil {
		hm.mu.Unlock()
		return
	}
	hm.processorCtx, hm.processorCancel = context.WithCancel(ctx)
	hm.mu.Unlock()

	go func() {
		defer func() {
			hm.mu.Lock()
			hm.processorCancel = nil
			hm.processorCtx = nil
			close(hm.processorDone)
			hm.mu.Unlock()
		}()

		moduleReady := false
		for !moduleReady {
			select {
			case <-hm.processorCtx.Done():
				return
			case <-hm.processorStop:
				return
			case <-time.After(50 * time.Millisecond):
				if h.Module != nil {
					moduleReady = true
				}
			}
		}

		for {
			select {
			case <-hm.processorCtx.Done():
				slog.Info(i18n.Msg("HTTP processor: context cancelled"))
				return
			case <-hm.processorStop:
				slog.Info(i18n.Msg("HTTP processor: stopped"))
				return
			default:
				hm.processRequest(hm.processorCtx, h)
			}
		}
	}()
}

func (hm *httpManager) StopProcessor() {

	hm.mu.Lock()
	processorCancel := hm.processorCancel
	hm.mu.Unlock()

	if processorCancel == nil {
		return
	}

	processorCancel()

	select {
	case <-hm.processorStop:
	default:
		close(hm.processorStop)
	}

	<-hm.processorDone

	hm.mu.Lock()
	hm.processorStop = make(chan struct{})
	hm.processorDone = make(chan struct{})
	hm.mu.Unlock()
}

func (hm *httpManager) processRequest(ctx context.Context, h *host.Host) {

	if h.Module == nil {
		slog.Error(i18n.Msg("processRequest: module is not initialized"))
		return
	}

	dispatchFunc := h.Module.ExportedFunction("_dispatch")
	if dispatchFunc == nil {
		slog.Error(i18n.Msg("processRequest: _dispatch function not found"))
		return
	}

	_, callErr := dispatchFunc.Call(ctx)
	if callErr != nil {
		if errors.Is(callErr, context.Canceled) {
			return
		}
		var exitErr *sys.ExitError
		if errors.As(callErr, &exitErr) && (exitErr.ExitCode() == 0 || exitErr.ExitCode() == sys.ExitCodeContextCanceled) {
			return
		}
		slog.Error(i18n.Msg("processRequest: failed to call _dispatch"), "error", callErr)
		return
	}
}

func (reqCtx *RequestContext) readRequestBody(buf []byte) (n int, err error) {

	if reqCtx.BodyOffset >= len(reqCtx.BodyBuffer) {
		if reqCtx.BodyReader != nil {
			var readErr error
			reqCtx.BodyBuffer = make([]byte, 8192) // Буфер 8KB
			var bytesRead int
			if bytesRead, readErr = reqCtx.BodyReader.Read(reqCtx.BodyBuffer); readErr != nil && readErr != io.EOF {
				return 0, readErr
			}
			reqCtx.BodyBuffer = reqCtx.BodyBuffer[:bytesRead]
			reqCtx.BodyOffset = 0
		} else {
			return 0, io.EOF
		}
	}

	if reqCtx.BodyOffset >= len(reqCtx.BodyBuffer) {
		return 0, io.EOF
	}

	copyLen := len(reqCtx.BodyBuffer) - reqCtx.BodyOffset
	if copyLen > len(buf) {
		copyLen = len(buf)
	}

	copy(buf, reqCtx.BodyBuffer[reqCtx.BodyOffset:reqCtx.BodyOffset+copyLen])
	reqCtx.BodyOffset += copyLen

	return copyLen, nil
}

// writeRequestInfo: формат method_len(4)+method+url_len(4)+url+headers_count(4)+[key_len+key+value_len+value]*.
func writeRequestInfo(h *host.Host, reqCtx *RequestContext, infoBufPtr uint32, infoBufLen uint32) (bytesWritten uint32, err error) {

	req := reqCtx.Request

	methodBytes := []byte(req.Method)
	urlBytes := []byte(req.URL.String())

	headersSize := 0
	for key, values := range req.Header {
		keyBytes := []byte(key)
		for _, value := range values {
			valueBytes := []byte(value)
			headersSize += 4 + len(keyBytes) + 4 + len(valueBytes)
		}
	}

	totalSize := 4 + len(methodBytes) + 4 + len(urlBytes) + 4 + headersSize

	var totalSizeU32 uint32
	if totalSizeU32, err = safeIntToUint32(totalSize); err != nil {
		return 0, err
	}

	if totalSizeU32 > infoBufLen {
		return 0, fmt.Errorf(i18n.Msg("buffer too small: need %d, have %d"), totalSize, infoBufLen)
	}

	mem := h.Module.Memory()
	if mem == nil {
		return 0, errors.New(i18n.Msg("memory is not available"))
	}

	buf := make([]byte, 0, totalSize)

	methodLen := len(methodBytes)
	if methodLen > int(^uint32(0)) {
		return 0, errors.New(i18n.Msg("method length exceeds uint32 maximum"))
	}
	methodLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(methodLenBytes, uint32(methodLen))
	buf = append(buf, methodLenBytes...)
	buf = append(buf, methodBytes...)

	urlLen := len(urlBytes)
	if urlLen > int(^uint32(0)) {
		return 0, errors.New(i18n.Msg("url length exceeds uint32 maximum"))
	}
	urlLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(urlLenBytes, uint32(urlLen))
	buf = append(buf, urlLenBytes...)
	buf = append(buf, urlBytes...)

	headerCount := 0
	for _, values := range req.Header {
		valuesCount := len(values)
		if valuesCount > int(^uint32(0))-headerCount {
			return 0, errors.New(i18n.Msg("header count exceeds uint32 maximum"))
		}
		headerCount += valuesCount
	}
	var headerCountU32 uint32
	if headerCountU32, err = safeIntToUint32(headerCount); err != nil {
		return 0, err
	}
	headerCountBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(headerCountBytes, headerCountU32)
	buf = append(buf, headerCountBytes...)

	for key, values := range req.Header {
		keyBytes := []byte(key)
		keyLen := len(keyBytes)
		if keyLen > int(^uint32(0)) {
			return 0, errors.New(i18n.Msg("header key length exceeds uint32 maximum"))
		}
		for _, value := range values {
			valueBytes := []byte(value)
			valueLen := len(valueBytes)
			if valueLen > int(^uint32(0)) {
				return 0, errors.New(i18n.Msg("header value length exceeds uint32 maximum"))
			}

			keyLenBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(keyLenBytes, uint32(keyLen))
			buf = append(buf, keyLenBytes...)
			buf = append(buf, keyBytes...)

			valueLenBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(valueLenBytes, uint32(valueLen))
			buf = append(buf, valueLenBytes...)
			buf = append(buf, valueBytes...)
		}
	}

	if !mem.Write(infoBufPtr, buf) {
		return 0, errors.New(i18n.Msg("failed to write request info"))
	}

	bufLen := len(buf)
	if bufLen > int(^uint32(0)) {
		return 0, errors.New(i18n.Msg("buffer length exceeds uint32 maximum"))
	}

	return uint32(bufLen), nil
}

// hostListenAndServe: неблокирующая — возвращает управление сразу после запуска сервера; обработка в плагине через _dispatch.
func hostListenAndServe(ctx context.Context, h *host.Host, hm *httpManager, addrPtr uint32, addrLen uint32, handlerID uint64) (result uint64) {

	var err error
	var addrBytes []byte
	if addrBytes, err = memory.Read(h, addrPtr, addrLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	addr := string(addrBytes)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 30 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyReader := bufio.NewReader(r.Body)
			reqCtx := &RequestContext{
				Request:        r,
				ResponseWriter: w,
				DoneChan:       make(chan struct{}),
				BodyReader:     bodyReader,
				BodyBuffer:     nil,
				BodyOffset:     0,
				HandlerID:      handlerID,
			}

			requestID := hm.StoreRequest(reqCtx)
			hm.EnqueueRequest(requestID)
			<-reqCtx.DoneChan
		}),
	}

	serverID := hm.StoreServer(server, addr, nil)

	hm.mu.RLock()
	serverCount := len(hm.serverMap)
	hm.mu.RUnlock()
	if serverCount == 1 {
		hm.StartProcessor(ctx, h)
	}

	h.ActiveServers.Add(1)
	go func() {
		defer h.ActiveServers.Done()
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			slog.Error(i18n.Msg("HTTP server error"), "error", serveErr, "addr", addr, "serverID", serverID)
		}
		hm.DelServer(serverID)
	}()

	return serverID
}

func hostStopServer(ctx context.Context, h *host.Host, hm *httpManager, serverID uint64) (result uint64) {

	slog.Info(i18n.Msg("host_stop_server: stopping server"), "serverID", serverID)

	var err error
	var serverState *httpServerState
	if serverState, err = hm.GetServer(serverID); err != nil {
		slog.Error(i18n.Msg("host_stop_server: server not found"), "serverID", serverID, "error", err)
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("server id %d does not exist"), serverID))
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info(i18n.Msg("host_stop_server: calling Shutdown"), "serverID", serverID, "addr", serverState.Addr)
	if shutdownErr := serverState.Server.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error(i18n.Msg("host_stop_server: shutdown failed"), "serverID", serverID, "error", shutdownErr)
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to shutdown server: %w"), shutdownErr))
	}

	hm.DelServer(serverID)

	slog.Info(i18n.Msg("host_stop_server: server stopped successfully"), "serverID", serverID)
	return 0
}

// hostGetNextRequest вызывается из _dispatch в WASM; блокируется до появления запроса в очереди.
func hostGetNextRequest(ctx context.Context, h *host.Host, hm *httpManager, requestIDPtr uint32, handlerIDPtr uint32) (result uint64) {

	if h.Module == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("module is not initialized")))
	}

	var requestID uint64
	select {
	case requestID = <-hm.requestQueue:
	case <-ctx.Done():
		return writeError(ctx, h, errors.New(i18n.Msg("context cancelled while waiting for request")))
	}

	var err error
	var reqCtx *RequestContext
	if reqCtx, err = hm.GetRequest(requestID); err != nil {
		return writeError(ctx, h, err)
	}

	mem := h.Module.Memory()
	if mem == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	if requestID > uint64(^uint32(0)) {
		return writeError(ctx, h, errors.New(i18n.Msg("request id exceeds uint32 maximum")))
	}
	if !mem.WriteUint32Le(requestIDPtr, uint32(requestID)) {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write request id")))
	}

	if reqCtx.HandlerID > uint64(^uint32(0)) {
		return writeError(ctx, h, errors.New(i18n.Msg("handler id exceeds uint32 maximum")))
	}
	if !mem.WriteUint32Le(handlerIDPtr, uint32(reqCtx.HandlerID)) {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write handler id")))
	}

	return 0
}

func hostGetRequestInfo(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, infoBufPtr uint32, infoBufLen uint32) (result uint64) {

	var err error
	var reqCtx *RequestContext
	if reqCtx, err = hm.GetRequest(requestID); err != nil {
		return writeError(ctx, h, err)
	}

	var bytesWritten uint32
	if bytesWritten, err = writeRequestInfo(h, reqCtx, infoBufPtr, infoBufLen); err != nil {
		return writeError(ctx, h, err)
	}

	return uint64(bytesWritten)
}

func hostReadRequestBody(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, bufPtr uint32, bufLen uint32) (result uint64) {

	var err error
	var reqCtx *RequestContext
	if reqCtx, err = hm.GetRequest(requestID); err != nil {
		return writeError(ctx, h, err)
	}

	buf := make([]byte, bufLen)
	var bytesRead int
	if bytesRead, err = reqCtx.readRequestBody(buf); err != nil && err != io.EOF {
		return writeError(ctx, h, err)
	}

	if bytesRead == 0 {
		return 0
	}

	if err = memory.Write(h, bufPtr, buf[:bytesRead]); err != nil {
		return writeError(ctx, h, err)
	}

	if bytesRead < 0 {
		return writeError(ctx, h, errors.New(i18n.Msg("negative bytes read")))
	}
	return uint64(bytesRead)
}

// hostWriteResponseHeaders: WriteHeader должен быть вызван после установки заголовков (требование net/http).
func hostWriteResponseHeaders(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, statusCode int32, headersPtr uint32, headersLen uint32) (result uint64) {

	var err error
	var reqCtx *RequestContext
	if reqCtx, err = hm.GetRequest(requestID); err != nil {
		return writeError(ctx, h, err)
	}

	if headersLen > 0 {
		var headersBytes []byte
		if headersBytes, err = memory.Read(h, headersPtr, headersLen); err != nil {
			return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read headers: %w"), err))
		}

		offset := 0
		for offset < len(headersBytes) {
			if offset+4 > len(headersBytes) {
				break
			}
			keyLen := int(binary.LittleEndian.Uint32(headersBytes[offset : offset+4]))
			offset += 4

			if offset+keyLen > len(headersBytes) {
				break
			}
			key := string(headersBytes[offset : offset+keyLen])
			offset += keyLen

			if offset+4 > len(headersBytes) {
				break
			}
			valueLen := int(binary.LittleEndian.Uint32(headersBytes[offset : offset+4]))
			offset += 4

			if offset+valueLen > len(headersBytes) {
				break
			}
			value := string(headersBytes[offset : offset+valueLen])
			offset += valueLen

			reqCtx.ResponseWriter.Header().Set(key, value)
		}
	}

	reqCtx.ResponseWriter.WriteHeader(int(statusCode))

	return 0
}

func hostWriteResponseBody(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, dataPtr uint32, dataLen uint32) (result uint64) {

	var err error
	var reqCtx *RequestContext
	if reqCtx, err = hm.GetRequest(requestID); err != nil {
		return writeError(ctx, h, err)
	}

	var data []byte
	if data, err = memory.Read(h, dataPtr, dataLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read response body: %w"), err))
	}

	var bytesWritten int
	if bytesWritten, err = reqCtx.ResponseWriter.Write(data); err != nil {
		return writeError(ctx, h, err)
	}

	if bytesWritten < 0 {
		return writeError(ctx, h, errors.New(i18n.Msg("negative bytes written")))
	}
	return uint64(bytesWritten)
}

// hostFinishRequest: close(DoneChan) разблокирует горутину сервера, ожидающую завершения обработки.
func hostFinishRequest(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64) (result uint64) {

	var err error
	var reqCtx *RequestContext
	if reqCtx, err = hm.GetRequest(requestID); err != nil {
		return writeError(ctx, h, err)
	}

	hm.DelRequest(requestID)
	close(reqCtx.DoneChan)

	return 0
}

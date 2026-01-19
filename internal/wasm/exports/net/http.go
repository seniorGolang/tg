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

	handler, ok := hm.handlerMap[handlerID]
	if !ok {
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

	state, ok := hm.serverMap[serverID]
	if !ok {
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

	ctx, ok := hm.requestMap[requestID]
	if !ok {
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

// StopProcessor останавливает WASM-процессор.
func (hm *httpManager) StopProcessor() {

	hm.mu.Lock()
	processorCancel := hm.processorCancel
	hm.mu.Unlock()

	if processorCancel == nil {
		return // Процессор не запущен
	}

	// Отменяем контекст процессора
	processorCancel()

	// Закрываем канал остановки (только один раз)
	select {
	case <-hm.processorStop:
		// Уже закрыт
	default:
		close(hm.processorStop)
	}

	// Ждем завершения процессора
	<-hm.processorDone

	// Пересоздаем каналы для следующего запуска
	hm.mu.Lock()
	hm.processorStop = make(chan struct{})
	hm.processorDone = make(chan struct{})
	hm.mu.Unlock()
}

// processRequest обрабатывает один запрос, вызывая _dispatch в WASM.
func (hm *httpManager) processRequest(ctx context.Context, h *host.Host) {

	// Проверяем, что модуль готов
	if h.Module == nil {
		slog.Error(i18n.Msg("processRequest: module is not initialized"))
		return
	}

	// Вызываем экспортируемую функцию _dispatch в WASM
	// _dispatch сама вызовет host_get_next_request для получения request_id и handler_id
	dispatchFunc := h.Module.ExportedFunction("_dispatch")
	if dispatchFunc == nil {
		slog.Error(i18n.Msg("processRequest: _dispatch function not found"))
		return
	}

	// Вызываем _dispatch (без параметров)
	// _dispatch будет блокировать в host_get_next_request до получения запроса из очереди
	_, callErr := dispatchFunc.Call(ctx)
	if callErr != nil {
		slog.Error(i18n.Msg("processRequest: failed to call _dispatch"), "error", callErr)
		return
	}
}

func (reqCtx *RequestContext) readRequestBody(buf []byte) (n int, err error) {

	if reqCtx.BodyOffset >= len(reqCtx.BodyBuffer) {
		// Читаем следующий чанк из BodyReader
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

	// Копируем данные из буфера в buf
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

	// Подготавливаем данные
	methodBytes := []byte(req.Method)
	urlBytes := []byte(req.URL.String())

	// Подсчитываем размер заголовков
	headersSize := 0
	for key, values := range req.Header {
		keyBytes := []byte(key)
		for _, value := range values {
			valueBytes := []byte(value)
			headersSize += 4 + len(keyBytes) + 4 + len(valueBytes) // key_len + key + value_len + value
		}
	}

	// Общий размер: method_len(4) + method + url_len(4) + url + headers_count(4) + headers
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

	// Собираем все данные в один буфер
	buf := make([]byte, 0, totalSize)

	// method_len и method
	methodLen := len(methodBytes)
	if methodLen > int(^uint32(0)) {
		return 0, errors.New(i18n.Msg("method length exceeds uint32 maximum"))
	}
	methodLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(methodLenBytes, uint32(methodLen))
	buf = append(buf, methodLenBytes...)
	buf = append(buf, methodBytes...)

	// url_len и url
	urlLen := len(urlBytes)
	if urlLen > int(^uint32(0)) {
		return 0, errors.New(i18n.Msg("url length exceeds uint32 maximum"))
	}
	urlLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(urlLenBytes, uint32(urlLen))
	buf = append(buf, urlLenBytes...)
	buf = append(buf, urlBytes...)

	// headers_count
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

	// Заголовки
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

			// key_len и key
			keyLenBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(keyLenBytes, uint32(keyLen))
			buf = append(buf, keyLenBytes...)
			buf = append(buf, keyBytes...)

			// value_len и value
			valueLenBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(valueLenBytes, uint32(valueLen))
			buf = append(buf, valueLenBytes...)
			buf = append(buf, valueBytes...)
		}
	}

	// Записываем весь буфер в память
	if !mem.Write(infoBufPtr, buf) {
		return 0, errors.New(i18n.Msg("failed to write request info"))
	}

	bufLen := len(buf)
	if bufLen > int(^uint32(0)) {
		return 0, errors.New(i18n.Msg("buffer length exceeds uint32 maximum"))
	}

	return uint32(bufLen), nil
}

// hostListenAndServe регистрирует обработчик и запускает HTTP сервер на хосте.
// Неблокирующая функция - возвращает управление сразу после запуска сервера.
func hostListenAndServe(ctx context.Context, h *host.Host, hm *httpManager, addrPtr uint32, addrLen uint32, handlerID uint64) (result uint64) {

	// Читаем адрес из памяти
	var addrBytes []byte
	var err error
	if addrBytes, err = memory.Read(h, addrPtr, addrLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	addr := string(addrBytes)

	// Создаём HTTP сервер
	// Handler не нужен на хосте - обработка происходит в плагине через _dispatch
	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 30 * time.Second, // Защита от Slowloris атаки
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Создаём контекст запроса
			bodyReader := bufio.NewReader(r.Body)
			reqCtx := &RequestContext{
				Request:        r,
				ResponseWriter: w,
				DoneChan:       make(chan struct{}),
				BodyReader:     bodyReader,
				BodyBuffer:     nil,
				BodyOffset:     0,
				HandlerID:      handlerID, // Сохраняем handlerID в контексте
			}

			// Сохраняем контекст и получаем request_id
			requestID := hm.StoreRequest(reqCtx)

			// Помещаем request_id в очередь
			hm.EnqueueRequest(requestID)

			// Ждём завершения обработки
			<-reqCtx.DoneChan
		}),
	}

	// Сохраняем сервер (handler не нужен на хосте)
	serverID := hm.StoreServer(server, addr, nil)

	// Запускаем процессор при запуске первого сервера
	hm.mu.RLock()
	serverCount := len(hm.serverMap)
	hm.mu.RUnlock()
	if serverCount == 1 {
		// Это первый сервер - запускаем процессор
		hm.StartProcessor(ctx, h)
	}

	// Запускаем сервер в отдельной горутине
	h.ActiveServers.Add(1)
	go func() {
		defer h.ActiveServers.Done()
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			slog.Error(i18n.Msg("HTTP server error"), "error", serveErr, "addr", addr, "serverID", serverID)
		}
		hm.DelServer(serverID)
	}()

	// Возвращаем serverID в формате: младшие 32 бита = serverID, старшие 32 бита = 0 (успех)
	return serverID
}

// hostStopServer останавливает HTTP сервер по его ID.
func hostStopServer(ctx context.Context, h *host.Host, hm *httpManager, serverID uint64) (result uint64) {

	slog.Info(i18n.Msg("host_stop_server: stopping server"), "serverID", serverID)

	// Получаем сервер по ID
	serverState, err := hm.GetServer(serverID)
	if err != nil {
		slog.Error(i18n.Msg("host_stop_server: server not found"), "serverID", serverID, "error", err)
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("server id %d does not exist"), serverID))
	}

	// Останавливаем сервер через Shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info(i18n.Msg("host_stop_server: calling Shutdown"), "serverID", serverID, "addr", serverState.Addr)
	if shutdownErr := serverState.Server.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error(i18n.Msg("host_stop_server: shutdown failed"), "serverID", serverID, "error", shutdownErr)
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to shutdown server: %w"), shutdownErr))
	}

	// Удаляем сервер из карты
	hm.DelServer(serverID)

	slog.Info(i18n.Msg("host_stop_server: server stopped successfully"), "serverID", serverID)
	return 0
}

// hostGetNextRequest вызывается из _dispatch в WASM; блокируется до появления запроса в очереди.
func hostGetNextRequest(ctx context.Context, h *host.Host, hm *httpManager, requestIDPtr uint32, handlerIDPtr uint32) (result uint64) {

	// Проверяем, что модуль готов
	if h.Module == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("module is not initialized")))
	}

	var requestID uint64
	select {
	case requestID = <-hm.requestQueue:
		// Получили request_id из очереди
	case <-ctx.Done():
		return writeError(ctx, h, errors.New(i18n.Msg("context cancelled while waiting for request")))
	}

	// Получаем контекст запроса
	reqCtx, err := hm.GetRequest(requestID)
	if err != nil {
		return writeError(ctx, h, err)
	}

	mem := h.Module.Memory()
	if mem == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	// Записываем request_id (проверяем переполнение)
	if requestID > uint64(^uint32(0)) {
		return writeError(ctx, h, errors.New(i18n.Msg("request id exceeds uint32 maximum")))
	}
	if !mem.WriteUint32Le(requestIDPtr, uint32(requestID)) {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write request id")))
	}

	// Записываем handler_id из контекста запроса (проверяем переполнение)
	if reqCtx.HandlerID > uint64(^uint32(0)) {
		return writeError(ctx, h, errors.New(i18n.Msg("handler id exceeds uint32 maximum")))
	}
	if !mem.WriteUint32Le(handlerIDPtr, uint32(reqCtx.HandlerID)) {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write handler id")))
	}

	return 0
}

func hostGetRequestInfo(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, infoBufPtr uint32, infoBufLen uint32) (result uint64) {

	// Получаем контекст запроса
	reqCtx, err := hm.GetRequest(requestID)
	if err != nil {
		return writeError(ctx, h, err)
	}

	// Записываем информацию о запросе
	var bytesWritten uint32
	if bytesWritten, err = writeRequestInfo(h, reqCtx, infoBufPtr, infoBufLen); err != nil {
		return writeError(ctx, h, err)
	}

	// Возвращаем количество записанных байт
	return uint64(bytesWritten)
}

func hostReadRequestBody(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, bufPtr uint32, bufLen uint32) (result uint64) {

	// Получаем контекст запроса
	reqCtx, err := hm.GetRequest(requestID)
	if err != nil {
		return writeError(ctx, h, err)
	}

	// Читаем чанк данных
	buf := make([]byte, bufLen)
	var bytesRead int
	if bytesRead, err = reqCtx.readRequestBody(buf); err != nil && err != io.EOF {
		return writeError(ctx, h, err)
	}

	if bytesRead == 0 {
		// Тело закончилось
		return 0
	}

	// Записываем данные в WASM память
	if err = memory.Write(h, bufPtr, buf[:bytesRead]); err != nil {
		return writeError(ctx, h, err)
	}

	// Проверяем переполнение при конвертации int -> uint64
	if bytesRead < 0 {
		return writeError(ctx, h, errors.New(i18n.Msg("negative bytes read")))
	}
	return uint64(bytesRead)
}

func hostWriteResponseHeaders(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, statusCode int32, headersPtr uint32, headersLen uint32) (result uint64) {

	// Получаем контекст запроса
	reqCtx, err := hm.GetRequest(requestID)
	if err != nil {
		return writeError(ctx, h, err)
	}

	// Читаем заголовки из памяти (если есть)
	if headersLen > 0 {
		var headersBytes []byte
		if headersBytes, err = memory.Read(h, headersPtr, headersLen); err != nil {
			return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read headers: %w"), err))
		}

		// Парсим заголовки (формат: key_len(4) + key + value_len(4) + value повторяется)
		offset := 0
		for offset < len(headersBytes) {
			// Читаем key_len
			if offset+4 > len(headersBytes) {
				break
			}
			keyLen := int(binary.LittleEndian.Uint32(headersBytes[offset : offset+4]))
			offset += 4

			// Читаем key
			if offset+keyLen > len(headersBytes) {
				break
			}
			key := string(headersBytes[offset : offset+keyLen])
			offset += keyLen

			// Читаем value_len
			if offset+4 > len(headersBytes) {
				break
			}
			valueLen := int(binary.LittleEndian.Uint32(headersBytes[offset : offset+4]))
			offset += 4

			// Читаем value
			if offset+valueLen > len(headersBytes) {
				break
			}
			value := string(headersBytes[offset : offset+valueLen])
			offset += valueLen

			reqCtx.ResponseWriter.Header().Set(key, value)
		}
	}

	// Устанавливаем статус-код (WriteHeader должен быть вызван после установки заголовков)
	reqCtx.ResponseWriter.WriteHeader(int(statusCode))

	return 0
}

// hostWriteResponseBody пишет чанк тела ответа.
func hostWriteResponseBody(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64, dataPtr uint32, dataLen uint32) (result uint64) {

	// Получаем контекст запроса
	reqCtx, err := hm.GetRequest(requestID)
	if err != nil {
		return writeError(ctx, h, err)
	}

	// Читаем данные из WASM памяти
	var data []byte
	if data, err = memory.Read(h, dataPtr, dataLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read response body: %w"), err))
	}

	// Записываем данные в ResponseWriter
	var bytesWritten int
	if bytesWritten, err = reqCtx.ResponseWriter.Write(data); err != nil {
		return writeError(ctx, h, err)
	}

	// Проверяем переполнение при конвертации int -> uint64
	if bytesWritten < 0 {
		return writeError(ctx, h, errors.New(i18n.Msg("negative bytes written")))
	}
	return uint64(bytesWritten)
}

// hostFinishRequest завершает обработку запроса.
func hostFinishRequest(ctx context.Context, h *host.Host, hm *httpManager, requestID uint64) (result uint64) {

	// Получаем контекст запроса
	reqCtx, err := hm.GetRequest(requestID)
	if err != nil {
		return writeError(ctx, h, err)
	}

	// Удаляем контекст
	hm.DelRequest(requestID)

	// Сигнализируем о завершении обработки (разблокируем горутину сервера)
	close(reqCtx.DoneChan)

	return 0
}

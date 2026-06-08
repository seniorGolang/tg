// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package host

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

// CallRequest представляет запрос на вызов WASM функции через глобальный канал.
// Все аргументы передаются через указатели на память.
type CallRequest struct {
	// FunctionName - имя вызываемой WASM функции
	FunctionName string

	// Data - сериализованные данные для передачи в WASM память
	// Эти данные будут записаны в память перед вызовом функции
	Data []byte

	// ResultChan - канал для получения результата вызова
	// Результат будет содержать указатель и размер в памяти, или ошибку
	ResultChan chan CallResult
}

// CallResult представляет результат вызова WASM функции.
type CallResult struct {
	// ResultPtr - указатель на результат в WASM памяти (верхние 32 бита)
	// ResultSize - размер результата в WASM памяти (нижние 32 бита, 31-й бит - флаг ошибки)
	Result uint64

	// Error - ошибка выполнения вызова (если была)
	Error error
}

// CallChannel управляет глобальным каналом для вызовов WASM функций.
type CallChannel struct {
	// callChan - глобальный канал для всех вызовов WASM функций из хоста
	// Буферизованный канал фиксированного размера для предотвращения блокировок
	callChan chan *CallRequest

	// ctx - контекст для управления жизненным циклом обработчика
	ctx context.Context

	// cancel - функция отмены контекста
	cancel context.CancelFunc

	closeOnce sync.Once
	mu        sync.RWMutex
	closed    bool
}

// NewCallChannel запускает горутину-обработчик для последовательной обработки вызовов WASM.
func NewCallChannel(ctx context.Context, h *Host) (channel *CallChannel) {

	channelCtx, cancel := context.WithCancel(ctx)

	cc := &CallChannel{
		callChan: make(chan *CallRequest, 10),
		ctx:      channelCtx,
		cancel:   cancel,
	}

	go cc.processor(h)

	return cc
}

// processor последовательно обрабатывает вызовы из канала (однопоточный гость).
func (cc *CallChannel) processor(h *Host) {

	for {
		select {
		case <-cc.ctx.Done():
			return

		case req, ok := <-cc.callChan:
			if !ok {
				return
			}
			if req == nil {
				continue
			}
			result := cc.processCall(cc.ctx, h, req)
			req.ResultChan <- result
		}
	}
}

// processCall выполняет один вызов: выделение памяти, запись данных, вызов функции, чтение результата, освобождение памяти.
func (cc *CallChannel) processCall(ctx context.Context, h *Host, req *CallRequest) (result CallResult) {

	funcPtr := h.Module.ExportedFunction(req.FunctionName)
	if funcPtr == nil {
		return CallResult{
			Error: fmt.Errorf(i18n.Msg("function %s not found in WASM module"), req.FunctionName),
		}
	}

	var requestPtr uint32
	var err error
	if len(req.Data) > 0 {
		if requestPtr, err = allocateMemory(ctx, h, uint64(len(req.Data))); err != nil {
			return CallResult{
				Error: fmt.Errorf(i18n.Msg("failed to allocate memory for request: %w"), err),
			}
		}

		if err = writeMemory(h, requestPtr, req.Data); err != nil {
			freeMemory(ctx, h, uint64(requestPtr))
			return CallResult{
				Error: fmt.Errorf(i18n.Msg("failed to write request data to memory: %w"), err),
			}
		}
	}

	var results []uint64
	if len(req.Data) > 0 {
		results, err = funcPtr.Call(ctx, uint64(requestPtr), uint64(len(req.Data)))
	} else {
		results, err = funcPtr.Call(ctx)
	}

	if len(req.Data) > 0 {
		freeMemory(ctx, h, uint64(requestPtr))
	}

	if err != nil {
		return CallResult{
			Error: fmt.Errorf(i18n.Msg("failed to call function %s: %w"), req.FunctionName, err),
		}
	}

	if len(results) == 0 {
		return CallResult{
			Result: 0,
		}
	}

	return CallResult{
		Result: results[0],
	}
}

// Call помещает вызов WASM функции в глобальный канал.
// Возвращает канал для получения результата.
// Данные должны быть уже сериализованы в байты.
func (cc *CallChannel) Call(functionName string, data []byte) (resultChan <-chan CallResult) {

	ch := make(chan CallResult, 1)

	cc.mu.RLock()
	if cc.closed {
		cc.mu.RUnlock()
		ch <- CallResult{
			Error: errors.New(i18n.Msg("call channel is closed")),
		}
		return ch
	}

	req := &CallRequest{
		FunctionName: functionName,
		Data:         data,
		ResultChan:   ch,
	}

	// Не блокируем вызывающего при переполнении канала: callback (например, из сетевого слоя)
	// не должен ждать освобождения очереди, иначе возможна взаимная блокировка с processor.
	select {
	case cc.callChan <- req:
	case <-cc.ctx.Done():
		ch <- CallResult{
			Error: errors.New(i18n.Msg("call channel is closed")),
		}
	default:
		ch <- CallResult{
			Error: errors.New(i18n.Msg("call channel is full: too many pending calls")),
		}
	}
	cc.mu.RUnlock()

	return ch
}

func (cc *CallChannel) Close() {

	cc.closeOnce.Do(func() {
		cc.mu.Lock()
		cc.closed = true
		cc.mu.Unlock()

		// Риск: close(callChan) делает чтение всегда готовым и может дать nil-запрос,
		// а конкурентный send получит panic. Для остановки processor достаточно cancel:
		// context.CancelFunc потокобезопасен и последующие вызовы ничего не делают.
		// https://pkg.go.dev/context#CancelFunc
		cc.cancel()
	})
}

// CallWithUint64 помещает вызов WASM функции с uint64 аргументом в канал.
// uint64 сериализуется в 8 байт (binary.LittleEndian).
func (cc *CallChannel) CallWithUint64(functionName string, value uint64) (resultChan <-chan CallResult) {

	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, value)

	return cc.Call(functionName, data)
}

// allocateMemory выделяет память в WASM модуле через malloc.
func allocateMemory(ctx context.Context, h *Host, size uint64) (ptr uint32, err error) {

	if h.Malloc == nil {
		return 0, errors.New(i18n.Msg("malloc function is not available"))
	}

	if size == 0 {
		return 0, nil
	}

	var results []uint64
	if results, err = h.Malloc.Call(ctx, size); err != nil {
		return 0, fmt.Errorf(i18n.Msg("failed to allocate memory: %w"), err)
	}

	if len(results) == 0 {
		return 0, errors.New(i18n.Msg("malloc returned no results"))
	}

	allocatedPtr := results[0]
	if allocatedPtr > uint64(^uint32(0)) {
		return 0, errors.New(i18n.Msg("allocated pointer too large for uint32"))
	}

	return uint32(allocatedPtr), nil
}

// freeMemory освобождает память в WASM модуле через free.
func freeMemory(ctx context.Context, h *Host, ptr uint64) {

	if h.Free == nil {
		return
	}

	if ptr == 0 {
		return
	}

	_, _ = h.Free.Call(ctx, ptr)
}

func writeMemory(h *Host, ptr uint32, data []byte) (err error) {

	if h.Module == nil {
		return errors.New(i18n.Msg("module is not available"))
	}

	mem := h.Module.Memory()
	if mem == nil {
		return errors.New(i18n.Msg("memory is not available"))
	}

	if !mem.Write(ptr, data) {
		return fmt.Errorf(i18n.Msg("failed to write data to memory at ptr=%d, size=%d"), ptr, len(data))
	}

	return
}

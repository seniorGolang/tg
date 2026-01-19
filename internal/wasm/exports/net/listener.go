// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

func listenerListen(ctx context.Context, h *host.Host, nm *netManager, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, listenerIDPtr uint32) (result uint64) {

	var networkBytes []byte
	var err error
	if networkBytes, err = memory.Read(h, networkPtr, networkLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read network: %w"), err))
	}

	var addressBytes []byte
	if addressBytes, err = memory.Read(h, addressPtr, addressLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	network := string(networkBytes)
	address := string(addressBytes)

	var listener net.Listener
	if listener, err = net.Listen(network, address); err != nil {
		return writeError(ctx, h, err)
	}

	listenerID := nm.StoreListener(listener)

	if listenerID > uint64(^uint32(0)) {
		nm.DelListener(listenerID)
		return writeError(ctx, h, errors.New(i18n.Msg("listener id too large")))
	}

	if h.Module.Memory() == nil {
		nm.DelListener(listenerID)
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	if !h.Module.Memory().WriteUint32Le(listenerIDPtr, uint32(listenerID)) { //nolint:gosec // проверка на переполнение выполнена выше
		nm.DelListener(listenerID)
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write listener id")))
	}

	return 0
}

// ListenerAccept принимает новое соединение от слушателя.
func listenerAccept(ctx context.Context, h *host.Host, nm *netManager, listenerID uint64, connIDPtr uint32) (result uint64) {

	var listener net.Listener
	var err error
	if listener, err = nm.GetListener(listenerID); err != nil {
		return writeError(ctx, h, err)
	}

	var conn net.Conn
	if conn, err = listener.Accept(); err != nil {
		return writeError(ctx, h, err)
	}

	connID := nm.StoreConnWithStream(ctx, h, conn)

	if h.Module.Memory() == nil {
		nm.DelConn(connID)
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	if connID > uint64(^uint32(0)) {
		nm.DelConn(connID)
		return writeError(ctx, h, errors.New(i18n.Msg("connection id too large")))
	}

	if !h.Module.Memory().WriteUint32Le(connIDPtr, uint32(connID)) {
		nm.DelConn(connID)
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write connection id")))
	}

	// Вызываем callback on_new_connection через глобальный канал (если установлен)
	// Согласно WASM_CALL_CHANNEL_ARCHITECTURE.md, callback вызывается через канал вызовов
	nm.listenerLock.RLock()
	callback := nm.onNewConnectionCallback
	nm.listenerLock.RUnlock()

	if callback != nil {
		// Вызываем callback асинхронно, чтобы не блокировать host функцию
		// Callback будет обработан через глобальный канал вызовов
		go callback(connID)
	}

	return 0
}

func listenerClose(ctx context.Context, h *host.Host, nm *netManager, listenerID uint64) (result uint64) {

	var listener net.Listener
	var err error
	if listener, err = nm.GetListener(listenerID); err != nil {
		return writeError(ctx, h, err)
	}

	nm.DelListener(listenerID)

	if err = listener.Close(); err != nil {
		return writeError(ctx, h, err)
	}

	return 0
}

// ListenerServeStart запускает цикл listener.Accept() на хосте и передает соединения в плагин через callback.
// listenerID - ID listener'а, созданного через listener_listen
// callbackFuncPtr - указатель на имя экспортированной WASM функции для обработки соединений
// callbackFuncLen - длина имени функции
func listenerServeStart(ctx context.Context, h *host.Host, nm *netManager, listenerID uint64, callbackFuncPtr uint32, callbackFuncLen uint32) (result uint64) {

	// Получаем listener по listenerID
	var listener net.Listener
	var err error
	if listener, err = nm.GetListener(listenerID); err != nil {
		return writeError(ctx, h, err)
	}

	// Читаем имя callback функции из памяти
	var callbackNameBytes []byte
	if callbackNameBytes, err = memory.Read(h, callbackFuncPtr, callbackFuncLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read callback function name: %w"), err))
	}

	callbackName := string(callbackNameBytes)

	// Добавляем listener в ActiveListeners WaitGroup
	h.ActiveListeners.Add(1)

	// Запускаем горутину с циклом listener.Accept()
	go func() {
		defer h.ActiveListeners.Done()

		for {
			// Принимаем новое соединение
			var conn net.Conn
			var acceptErr error
			if conn, acceptErr = listener.Accept(); acceptErr != nil {
				// Если listener закрыт, выходим из цикла
				// Это нормальное завершение работы
				return
			}

			// Сохраняем conn на хосте → получаем connID
			connID := nm.StoreConnWithStream(ctx, h, conn)

			// Сериализуем listenerID и connID в байты
			dataBytes := make([]byte, 16)
			binary.LittleEndian.PutUint64(dataBytes[0:8], listenerID)
			binary.LittleEndian.PutUint64(dataBytes[8:16], connID)

			// Помещаем вызов callback в глобальный канал с данными для выделения памяти
			// Callback будет вызван через глобальный канал вызовов
			// CallChannel сам проверит наличие функции и обработает ошибки
			resultChan := h.CallChannel.Call(callbackName, dataBytes)
			// Проверяем результат вызова асинхронно
			// Контекст гарантирует завершение горутины при отмене, что позволяет GC собрать неиспользуемый канал
			go func() {
				select {
				case result := <-resultChan:
					if result.Error != nil {
						// Логируем ошибку, но не прерываем цикл Accept
						// Ошибка может быть из-за отсутствия функции или других проблем
						slog.Warn(i18n.Msg("Failed to call callback"), "callback", callbackName, "listenerID", listenerID, "connID", connID, "error", result.Error)
					}
				case <-ctx.Done():
					// Контекст отменен - завершаем горутину, канал будет собран GC
					return
				}
			}()
		}
	}()

	// Возвращаем управление плагину немедленно (неблокирующая)
	return 0
}

// ListenerAddr: addrPtr — буфер для записи адреса (предварительно выделен).
// addrLenPtr - указатель на переменную, содержащую размер буфера (входной) и реальную длину адреса (выходной)
func listenerAddr(ctx context.Context, h *host.Host, nm *netManager, listenerID uint64, addrPtr uint32, addrLenPtr uint32) (result uint64) {

	var listener net.Listener
	var err error
	if listener, err = nm.GetListener(listenerID); err != nil {
		return writeError(ctx, h, err)
	}

	addr := listener.Addr().String()

	var mem api.Memory
	if mem = h.Module.Memory(); mem == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	// Читаем размер буфера из addrLenPtr (входной параметр)
	var lengthBytes []byte
	var ok bool
	if lengthBytes, ok = mem.Read(addrLenPtr, 4); !ok {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to read buffer size")))
	}

	bufferSize := binary.LittleEndian.Uint32(lengthBytes)

	addrBytes := []byte(addr)
	addrLen := len(addrBytes)

	if addrLen < 0 || addrLen > int(^uint32(0)) {
		return writeError(ctx, h, errors.New(i18n.Msg("address length out of range")))
	}

	// Записываем адрес в буфер (не более размера буфера)
	writeLen := addrLen
	if writeLen > int(bufferSize) {
		writeLen = int(bufferSize)
	}

	if writeLen > 0 {
		if !mem.Write(addrPtr, addrBytes[:writeLen]) {
			return writeError(ctx, h, errors.New(i18n.Msg("failed to write address")))
		}
	}

	// Записываем реальную длину адреса в addrLenPtr (выходной параметр)
	if !mem.WriteUint32Le(addrLenPtr, uint32(addrLen)) { //nolint:gosec // проверка на переполнение выполнена выше
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write address length")))
	}

	return 0
}

// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"context"
	"encoding/binary"
	"errors"
	"net"

	"github.com/tetratelabs/wazero/api"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
)

// ConnRemoteAddr: addrPtr — буфер для записи адреса (предварительно выделен).
// addrLenPtr - указатель на переменную, содержащую размер буфера (входной) и реальную длину адреса (выходной)
func connRemoteAddr(ctx context.Context, h *host.Host, nm *netManager, connID uint64, addrPtr uint32, addrLenPtr uint32) (result uint64) {

	var conn net.Conn
	var err error
	if conn, err = nm.GetConn(connID); err != nil {
		return writeError(ctx, h, err)
	}

	remoteAddr := conn.RemoteAddr().String()

	var mem api.Memory
	if mem = h.Module.Memory(); mem == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	var lengthBytes []byte
	var ok bool
	if lengthBytes, ok = mem.Read(addrLenPtr, 4); !ok {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to read buffer size")))
	}

	bufferSize := binary.LittleEndian.Uint32(lengthBytes)

	remoteAddrBytes := []byte(remoteAddr)
	remoteAddrLen := len(remoteAddrBytes)

	if remoteAddrLen < 0 || remoteAddrLen > int(^uint32(0)) {
		return writeError(ctx, h, errors.New(i18n.Msg("address length out of range")))
	}

	writeLen := remoteAddrLen
	if writeLen > int(bufferSize) {
		writeLen = int(bufferSize)
	}

	if writeLen > 0 {
		if !mem.Write(addrPtr, remoteAddrBytes[:writeLen]) {
			return writeError(ctx, h, errors.New(i18n.Msg("failed to write address")))
		}
	}

	if !mem.WriteUint32Le(addrLenPtr, uint32(remoteAddrLen)) { //nolint:gosec // проверка на переполнение выполнена выше
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write address length")))
	}

	return 0
}

// ConnLocalAddr: addrPtr — буфер для записи адреса (предварительно выделен).
// addrLenPtr - указатель на переменную, содержащую размер буфера (входной) и реальную длину адреса (выходной)
func connLocalAddr(ctx context.Context, h *host.Host, nm *netManager, connID uint64, addrPtr uint32, addrLenPtr uint32) (result uint64) {

	var conn net.Conn
	var err error
	if conn, err = nm.GetConn(connID); err != nil {
		return writeError(ctx, h, err)
	}

	localAddr := conn.LocalAddr().String()

	var mem api.Memory
	if mem = h.Module.Memory(); mem == nil {
		return writeError(ctx, h, errors.New(i18n.Msg("memory is not available")))
	}

	var lengthBytes []byte
	var ok bool
	if lengthBytes, ok = mem.Read(addrLenPtr, 4); !ok {
		return writeError(ctx, h, errors.New(i18n.Msg("failed to read buffer size")))
	}

	bufferSize := binary.LittleEndian.Uint32(lengthBytes)

	localAddrBytes := []byte(localAddr)
	localAddrLen := len(localAddrBytes)

	if localAddrLen < 0 || localAddrLen > int(^uint32(0)) {
		return writeError(ctx, h, errors.New(i18n.Msg("address length out of range")))
	}

	writeLen := localAddrLen
	if writeLen > int(bufferSize) {
		writeLen = int(bufferSize)
	}

	if writeLen > 0 {
		if !mem.Write(addrPtr, localAddrBytes[:writeLen]) {
			return writeError(ctx, h, errors.New(i18n.Msg("failed to write address")))
		}
	}

	if !mem.WriteUint32Le(addrLenPtr, uint32(localAddrLen)) { //nolint:gosec // проверка на переполнение выполнена выше
		return writeError(ctx, h, errors.New(i18n.Msg("failed to write address length")))
	}

	return 0
}

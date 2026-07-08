// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package memory

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/seniorGolang/tg/v3/internal/i18n"
)

func AllocateAndWrite(ctx context.Context, h Host, data []byte) (ptr uint32, size uint32, err error) {

	if len(data) == 0 {
		return 0, 0, nil
	}

	dataLen := len(data)
	if dataLen > int(^uint32(0)) {
		return 0, 0, fmt.Errorf(i18n.Msg("data size too large for uint32: %d"), dataLen)
	}

	size = uint32(dataLen)
	if ptr, err = Allocate(ctx, h, uint64(size)); err != nil {
		return 0, 0, fmt.Errorf(i18n.Msg("failed to allocate memory: %w"), err)
	}

	if err = Write(h, ptr, data); err != nil {
		Free(ctx, h, uint64(ptr))
		return 0, 0, fmt.Errorf(i18n.Msg("failed to write data: %w"), err)
	}

	return
}

func AllocateAndWriteObject[T any](ctx context.Context, h Host, v T) (ptr uint64, size uint64, err error) {

	var data []byte
	if data, err = json.Marshal(v); err != nil {
		return 0, 0, fmt.Errorf(i18n.Msg("failed to marshal JSON: %w"), err)
	}

	var ptr32, size32 uint32
	if ptr32, size32, err = AllocateAndWrite(ctx, h, data); err != nil {
		return 0, 0, err
	}

	return uint64(ptr32), uint64(size32), nil
}

func ReadAndUnmarshal[T any](h Host, ptr uint32, size uint32, v *T) (err error) {

	var data []byte
	if data, err = Read(h, ptr, size); err != nil {
		return fmt.Errorf(i18n.Msg("failed to read memory: %w"), err)
	}

	if err = json.Unmarshal(data, v); err != nil {
		return fmt.Errorf(i18n.Msg("failed to unmarshal JSON: %w"), err)
	}

	return nil
}

func WriteBytesToPtrSize(ctx context.Context, h Host, data []byte, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	dataLen := len(data)
	if dataLen == 0 {
		return 1
	}
	if dataLen > int(^uint32(0)) {
		return 1
	}
	dataSize := uint32(dataLen)

	var err error
	var ptr uint32
	if ptr, err = Allocate(ctx, h, uint64(dataSize)); err != nil {
		return 1
	}

	shouldFree := true
	defer func() {

		if shouldFree {
			Free(ctx, h, uint64(ptr))
		}
	}()

	if err = Write(h, ptr, data); err != nil {
		return 1
	}

	ptrBytes := make([]byte, 4)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ptrBytes, ptr)
	binary.LittleEndian.PutUint32(sizeBytes, dataSize)

	if err = Write(h, resultPtrPtr, ptrBytes); err != nil {
		return 1
	}
	if err = Write(h, resultSizePtr, sizeBytes); err != nil {
		return 1
	}

	shouldFree = false
	return 0
}

func WriteObjectToPtrSize[T any](ctx context.Context, h Host, v T, resultPtrPtr uint32, resultSizePtr uint32) (resultCode uint32) {

	var err error
	var data []byte
	if data, err = json.Marshal(v); err != nil {
		return 1
	}

	return WriteBytesToPtrSize(ctx, h, data, resultPtrPtr, resultSizePtr)
}

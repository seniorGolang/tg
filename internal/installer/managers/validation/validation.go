// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package validation

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/installer/managers"
)

// engine реализует ValidationEngine.
type engine struct{}

func NewEngine() managers.ValidationEngine {
	return &engine{}
}

const (
	bufferSize = 32 * 1024
)

func (e *engine) ValidateChecksum(ctx context.Context, filePath string, algorithm string, expected string) (err error) {

	var file *os.File
	if file, err = os.Open(filePath); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to open file: %w"), err)
		return
	}
	defer file.Close()

	var hasher hash.Hash
	algorithm = strings.ToLower(algorithm)

	switch algorithm {
	case "sha256":
		hasher = sha256.New()
	case "sha512":
		hasher = sha512.New()
	default:
		err = fmt.Errorf(i18n.Msg("Unsupported algorithm: %s"), algorithm)
		return
	}

	buffer := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		var n int
		var readErr error
		if n, readErr = file.Read(buffer); n > 0 {
			var writeErr error
			if _, writeErr = hasher.Write(buffer[:n]); writeErr != nil {
				err = fmt.Errorf(i18n.Msg("Failed to write to hash: %w"), writeErr)
				return
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			err = fmt.Errorf(i18n.Msg("Failed to read file: %w"), readErr)
			return
		}
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	expected = strings.TrimPrefix(expected, algorithm+":")
	expected = strings.ToLower(expected)

	if actual != expected {
		err = fmt.Errorf(i18n.Msg("Checksum mismatch: expected %s, got %s"), expected, actual)
		return
	}

	return
}

func (e *engine) ValidateSignature(ctx context.Context, filePath string, signature string) (err error) {

	err = errors.New(i18n.Msg("Digital signature validation not implemented"))
	return
}

func (e *engine) ValidateExecutable(ctx context.Context, filePath string) (err error) {

	var info os.FileInfo
	if info, err = os.Stat(filePath); err != nil {
		err = fmt.Errorf(i18n.Msg("Failed to get file information: %w"), err)
		return
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		err = errors.New(i18n.Msg("File is not executable"))
		return
	}

	return
}

func (e *engine) ValidateArchive(ctx context.Context, filePath string, format string) (err error) {

	var statErr error
	if _, statErr = os.Stat(filePath); statErr != nil {
		err = fmt.Errorf(i18n.Msg("File does not exist: %w"), statErr)
		return
	}

	return
}

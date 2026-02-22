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

type engine struct{}

func NewEngine() (eng managers.ValidationEngine) {
	return &engine{}
}

const (
	bufferSize = 32 * 1024
)

func (e *engine) ValidateChecksum(ctx context.Context, filePath string, algorithm string, expected string) (err error) {

	var file *os.File
	if file, err = os.Open(filePath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to open file: %w"), err)
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
		return fmt.Errorf(i18n.Msg("Unsupported algorithm: %s"), algorithm)
	}

	buffer := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var n int
		var readErr error
		if n, readErr = file.Read(buffer); n > 0 {
			var writeErr error
			if _, writeErr = hasher.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf(i18n.Msg("Failed to write to hash: %w"), writeErr)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf(i18n.Msg("Failed to read file: %w"), readErr)
		}
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	expected = strings.TrimPrefix(expected, algorithm+":")
	expected = strings.ToLower(expected)

	if actual != expected {
		return fmt.Errorf(i18n.Msg("Checksum mismatch: expected %s, got %s"), expected, actual)
	}

	return
}

func (e *engine) ValidateSignature(ctx context.Context, filePath string, signature string) (err error) {

	return errors.New(i18n.Msg("Digital signature validation not implemented"))
}

func (e *engine) ValidateExecutable(ctx context.Context, filePath string) (err error) {

	var info os.FileInfo
	if info, err = os.Stat(filePath); err != nil {
		return fmt.Errorf(i18n.Msg("Failed to get file information: %w"), err)
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		return errors.New(i18n.Msg("File is not executable"))
	}

	return
}

func (e *engine) ValidateArchive(ctx context.Context, filePath string, format string) (err error) {

	var statErr error
	if _, statErr = os.Stat(filePath); statErr != nil {
		return fmt.Errorf(i18n.Msg("File does not exist: %w"), statErr)
	}

	return
}

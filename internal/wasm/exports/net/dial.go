// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package net

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/seniorGolang/tg/v3/internal/i18n"
	"github.com/seniorGolang/tg/v3/internal/wasm/host"
	"github.com/seniorGolang/tg/v3/internal/wasm/memory"
)

// validateHost: wildcard (*.example.com), IP, CIDR.
// Также проверяет резолв доменов - если домен резолвится в разрешённый IP, подключение разрешается.
func validateHost(allowedHosts []string, address string) (err error) {

	if len(allowedHosts) == 0 {
		return errors.New(i18n.Msg("host is not allowed: plugin has no allowed hosts"))
	}

	host := address
	if hostPart, _, splitErr := net.SplitHostPort(address); splitErr == nil {
		host = hostPart
	}

	for _, allowed := range allowedHosts {
		if isHostAllowed(host, allowed) {
			return
		}
	}

	// Если хост - это не IP-адрес, пытаемся резолвить домен и проверить IP
	if hostIP := parseIP(host); hostIP == nil {
		if resolvedIPs, resolveErr := net.LookupIP(host); resolveErr == nil {
			for _, resolvedIP := range resolvedIPs {
				for _, allowed := range allowedHosts {
					if isIPAllowed(resolvedIP, allowed) {
						return
					}
				}
			}
		}
	}

	return fmt.Errorf(i18n.Msg("host %s is not allowed: not found in allowed hosts list"), host)
}

// isHostAllowed: точное совпадение, wildcard (*.example.com), CIDR для IP.
func isHostAllowed(host string, pattern string) (allowed bool) {

	if host == pattern {
		return true
	}

	hostIP := parseIP(host)
	if hostIP != nil {
		return isIPAllowed(hostIP, pattern)
	}

	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		if strings.HasSuffix(host, suffix) {
			// Это предотвращает совпадение "example.com" с паттерном "*.example.com"
			prefix := host[:len(host)-len(suffix)]
			if len(prefix) > 1 && prefix[0] == '.' {
				return true
			}
		}
	}

	return false
}

// parseIP парсит IP-адрес из строки, поддерживая IPv4 и IPv6.
// Для IPv6 адресов убирает квадратные скобки, если они есть.
func parseIP(host string) (ip net.IP) {

	if len(host) > 0 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}

	return net.ParseIP(host)
}

// isIPAllowed: точное совпадение или CIDR.
func isIPAllowed(ip net.IP, pattern string) (allowed bool) {

	if patternIP := parseIP(pattern); patternIP != nil {
		return ip.Equal(patternIP)
	}

	if _, ipNet, err := net.ParseCIDR(pattern); err == nil {
		return ipNet.Contains(ip)
	}

	return false
}

func connDial(ctx context.Context, h *host.Host, nm *netManager, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {

	var err error
	var networkBytes []byte
	if networkBytes, err = memory.Read(h, networkPtr, networkLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read network: %w"), err))
	}

	var addressBytes []byte
	if addressBytes, err = memory.Read(h, addressPtr, addressLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	network := string(networkBytes)
	address := string(addressBytes)

	if err = validateHost(h.Info.AllowedHosts, address); err != nil {
		return writeError(ctx, h, err)
	}

	var conn net.Conn
	if conn, err = net.Dial(network, address); err != nil {
		slog.Error(i18n.Msg("ConnDial: dial failed"), "error", err, "network", network, "address", address)
		return writeError(ctx, h, err)
	}

	if network == "tcp" {
		var tcpConn *net.TCPConn
		var ok bool
		if tcpConn, ok = conn.(*net.TCPConn); ok {
			_ = tcpConn.SetKeepAlive(true)
			_ = tcpConn.SetKeepAlivePeriod(time.Second * 3)
		}
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

	return 0
}

func connDialContext(ctx context.Context, h *host.Host, nm *netManager, deadline uint64, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {

	var err error
	var networkBytes []byte
	if networkBytes, err = memory.Read(h, networkPtr, networkLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read network: %w"), err))
	}

	var addressBytes []byte
	if addressBytes, err = memory.Read(h, addressPtr, addressLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	network := string(networkBytes)
	address := string(addressBytes)

	if err = validateHost(h.Info.AllowedHosts, address); err != nil {
		return writeError(ctx, h, err)
	}

	dialCtx := ctx
	if deadline > 0 {
		const maxInt64 = uint64(1<<63 - 1)
		if deadline > maxInt64 {
			return writeError(ctx, h, errors.New(i18n.Msg("deadline value too large")))
		}
		deadlineTime := time.Unix(0, int64(deadline))
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithDeadline(ctx, deadlineTime)
		defer cancel()
	}

	dialer := &net.Dialer{}
	var conn net.Conn
	if conn, err = dialer.DialContext(dialCtx, network, address); err != nil {
		slog.Error(i18n.Msg("ConnDialContext: dial failed"), "error", err, "network", network, "address", address)
		return writeError(ctx, h, err)
	}

	if network == "tcp" {
		var tcpConn *net.TCPConn
		var ok bool
		if tcpConn, ok = conn.(*net.TCPConn); ok {
			_ = tcpConn.SetKeepAlive(true)
			_ = tcpConn.SetKeepAlivePeriod(time.Second * 3)
		}
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

	return 0
}

func connDialTLS(ctx context.Context, h *host.Host, nm *netManager, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {

	var err error
	var networkBytes []byte
	if networkBytes, err = memory.Read(h, networkPtr, networkLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read network: %w"), err))
	}

	var addressBytes []byte
	if addressBytes, err = memory.Read(h, addressPtr, addressLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	network := string(networkBytes)
	address := string(addressBytes)

	if err = validateHost(h.Info.AllowedHosts, address); err != nil {
		return writeError(ctx, h, err)
	}

	dialer := &net.Dialer{}
	var tcpConn net.Conn
	if tcpConn, err = dialer.DialContext(ctx, network, address); err != nil {
		slog.Error(i18n.Msg("ConnDialTLS: dial failed"), "error", err, "network", network, "address", address)
		return writeError(ctx, h, err)
	}

	hostname := address
	var host string
	if host, _, err = net.SplitHostPort(address); err == nil {
		hostname = host
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         hostname,
		InsecureSkipVerify: h.TLSConfig.InsecureSkipVerify, //nolint:gosec // контролируется конфигурацией хоста
	}

	if !tlsCfg.InsecureSkipVerify {
		var systemRoots *x509.CertPool
		if systemRoots, err = x509.SystemCertPool(); err == nil && systemRoots != nil {
			tlsCfg.RootCAs = systemRoots
		}
	}

	tlsConn := tls.Client(tcpConn, tlsCfg)
	if err = tlsConn.HandshakeContext(ctx); err != nil {
		tcpConn.Close()
		return writeError(ctx, h, err)
	}

	conn := tlsConn

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

	return 0
}

func connDialTLSContext(ctx context.Context, h *host.Host, nm *netManager, deadline uint64, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, connIDPtr uint32) (result uint64) {

	var err error
	var networkBytes []byte
	if networkBytes, err = memory.Read(h, networkPtr, networkLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read network: %w"), err))
	}

	var addressBytes []byte
	if addressBytes, err = memory.Read(h, addressPtr, addressLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	network := string(networkBytes)
	address := string(addressBytes)

	if err = validateHost(h.Info.AllowedHosts, address); err != nil {
		return writeError(ctx, h, err)
	}

	dialCtx := ctx
	if deadline > 0 {
		const maxInt64 = uint64(1<<63 - 1)
		if deadline > maxInt64 {
			return writeError(ctx, h, errors.New(i18n.Msg("deadline too large")))
		}
		deadlineTime := time.Unix(0, int64(deadline))
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithDeadline(ctx, deadlineTime)
		defer cancel()
	}

	dialer := &net.Dialer{}
	var tcpConn net.Conn
	if tcpConn, err = dialer.DialContext(dialCtx, network, address); err != nil {
		slog.Error(i18n.Msg("ConnDialTLSContext: dial failed"), "error", err, "network", network, "address", address)
		return writeError(ctx, h, err)
	}

	hostname := address
	var host string
	if host, _, err = net.SplitHostPort(address); err == nil {
		hostname = host
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         hostname,
		InsecureSkipVerify: h.TLSConfig.InsecureSkipVerify, //nolint:gosec // контролируется конфигурацией хоста
	}

	if !tlsCfg.InsecureSkipVerify {
		var systemRoots *x509.CertPool
		if systemRoots, err = x509.SystemCertPool(); err == nil && systemRoots != nil {
			tlsCfg.RootCAs = systemRoots
		}
	}

	tlsConn := tls.Client(tcpConn, tlsCfg)
	if err = tlsConn.HandshakeContext(dialCtx); err != nil {
		slog.Error(i18n.Msg("ConnDialTLSContext: TLS handshake failed"), "error", err)
		tcpConn.Close()
		return writeError(ctx, h, err)
	}

	conn := tlsConn

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

	return 0
}

// tlsConfig представляет конфигурацию TLS для соединения.
type tlsConfig struct {
	MinVersion         string   `json:"min_version,omitempty"`          // "1.0", "1.1", "1.2", "1.3"
	MaxVersion         string   `json:"max_version,omitempty"`          // "1.0", "1.1", "1.2", "1.3"
	InsecureSkipVerify bool     `json:"insecure_skip_verify,omitempty"` // пропустить проверку сертификата
	ServerName         string   `json:"server_name,omitempty"`          // имя сервера для SNI
	CipherSuites       []string `json:"cipher_suites,omitempty"`        // список поддерживаемых cipher suites
}

func parseTLSConfig(cfg tlsConfig) (tlsCfg *tls.Config, err error) {

	tlsCfg = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.MinVersion != "" {
		switch cfg.MinVersion {
		case "1.0":
			tlsCfg.MinVersion = tls.VersionTLS10
		case "1.1":
			tlsCfg.MinVersion = tls.VersionTLS11
		case "1.2":
			tlsCfg.MinVersion = tls.VersionTLS12
		case "1.3":
			tlsCfg.MinVersion = tls.VersionTLS13
		default:
			return nil, fmt.Errorf(i18n.Msg("invalid min_version: %s"), cfg.MinVersion)
		}
	}

	if cfg.MaxVersion != "" {
		switch cfg.MaxVersion {
		case "1.0":
			tlsCfg.MaxVersion = tls.VersionTLS10
		case "1.1":
			tlsCfg.MaxVersion = tls.VersionTLS11
		case "1.2":
			tlsCfg.MaxVersion = tls.VersionTLS12
		case "1.3":
			tlsCfg.MaxVersion = tls.VersionTLS13
		default:
			return nil, fmt.Errorf(i18n.Msg("invalid max_version: %s"), cfg.MaxVersion)
		}
	}

	tlsCfg.InsecureSkipVerify = cfg.InsecureSkipVerify
	tlsCfg.ServerName = cfg.ServerName

	if len(cfg.CipherSuites) > 0 {
		cipherSuites := make([]uint16, 0, len(cfg.CipherSuites))
		for _, suite := range cfg.CipherSuites {
			var suiteID uint16
			switch suite {
			case "TLS_RSA_WITH_RC4_128_SHA":
				suiteID = tls.TLS_RSA_WITH_RC4_128_SHA
			case "TLS_RSA_WITH_3DES_EDE_CBC_SHA":
				suiteID = tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA
			case "TLS_RSA_WITH_AES_128_CBC_SHA":
				suiteID = tls.TLS_RSA_WITH_AES_128_CBC_SHA
			case "TLS_RSA_WITH_AES_256_CBC_SHA":
				suiteID = tls.TLS_RSA_WITH_AES_256_CBC_SHA
			case "TLS_RSA_WITH_AES_128_CBC_SHA256":
				suiteID = tls.TLS_RSA_WITH_AES_128_CBC_SHA256
			case "TLS_RSA_WITH_AES_128_GCM_SHA256":
				suiteID = tls.TLS_RSA_WITH_AES_128_GCM_SHA256
			case "TLS_RSA_WITH_AES_256_GCM_SHA384":
				suiteID = tls.TLS_RSA_WITH_AES_256_GCM_SHA384
			case "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":
				suiteID = tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA
			case "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":
				suiteID = tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
			case "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":
				suiteID = tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
			case "TLS_ECDHE_RSA_WITH_RC4_128_SHA":
				suiteID = tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA
			case "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":
				suiteID = tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA
			case "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":
				suiteID = tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
			case "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":
				suiteID = tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
			case "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":
				suiteID = tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256
			case "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":
				suiteID = tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256
			case "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":
				suiteID = tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
			case "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":
				suiteID = tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
			case "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":
				suiteID = tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
			case "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":
				suiteID = tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
			default:
				return nil, fmt.Errorf(i18n.Msg("unknown cipher suite: %s"), suite)
			}
			cipherSuites = append(cipherSuites, suiteID)
		}
		tlsCfg.CipherSuites = cipherSuites
	}

	return
}

// ConnDialTLSWithConfig: configPtr и configLen — JSON конфигурация TLS в памяти WASM.
func connDialTLSWithConfig(ctx context.Context, h *host.Host, nm *netManager, networkPtr uint32, networkLen uint32, addressPtr uint32, addressLen uint32, configPtr uint32, configLen uint32, connIDPtr uint32) (result uint64) {

	var err error
	var networkBytes []byte
	if networkBytes, err = memory.Read(h, networkPtr, networkLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read network: %w"), err))
	}

	var addressBytes []byte
	if addressBytes, err = memory.Read(h, addressPtr, addressLen); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to read address: %w"), err))
	}

	network := string(networkBytes)
	address := string(addressBytes)

	if err = validateHost(h.Info.AllowedHosts, address); err != nil {
		return writeError(ctx, h, err)
	}

	var tlsConfig tlsConfig
	if configLen > 0 {
		if err = memory.ReadAndUnmarshal(h, configPtr, configLen, &tlsConfig); err != nil {
			return writeError(ctx, h, fmt.Errorf(i18n.Msg("failed to parse TLS config: %w"), err))
		}
	}

	var tlsCfg *tls.Config
	if tlsCfg, err = parseTLSConfig(tlsConfig); err != nil {
		return writeError(ctx, h, fmt.Errorf(i18n.Msg("invalid TLS config: %w"), err))
	}

	var conn net.Conn
	if conn, err = tls.Dial(network, address, tlsCfg); err != nil {
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

	return 0
}

func connTLSHandshake(ctx context.Context, h *host.Host, nm *netManager, connID uint64) (result uint64) {

	var err error
	var conn net.Conn
	if conn, err = nm.GetConn(connID); err != nil {
		return writeError(ctx, h, err)
	}

	var ok bool
	var tlsConn *tls.Conn
	if tlsConn, ok = conn.(*tls.Conn); !ok {
		return writeError(ctx, h, errors.New(i18n.Msg("connection is not a TLS connection")))
	}

	var cancel context.CancelFunc
	var handshakeCtx context.Context
	handshakeCtx, cancel = context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if err = tlsConn.HandshakeContext(handshakeCtx); err != nil {
		return writeError(ctx, h, err)
	}

	return 0
}

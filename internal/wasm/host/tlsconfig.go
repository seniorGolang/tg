// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package host

// TLSConfig представляет конфигурацию TLS для host функций.
type TLSConfig struct {
	// InsecureSkipVerify отключает проверку сертификата сервера.
	// Используется для тестовых серверов с самоподписанными сертификатами.
	InsecureSkipVerify bool
}

func DefaultTLSConfig() (config TLSConfig) {

	return TLSConfig{
		InsecureSkipVerify: false,
	}
}

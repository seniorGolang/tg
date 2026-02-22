// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {

	tests := []struct {
		name           string
		spec           string
		expectedSource string
		expectedPkg    string
		expectedVer    string
		shouldError    bool
		errorContains  string
	}{
		// Валидные случаи из документации: URL:packageName@version
		{
			name:           "http://example.com/path:myapp@1.0.0",
			spec:           "http://example.com/path:myapp@1.0.0",
			expectedSource: "http://example.com/path",
			expectedPkg:    "myapp",
			expectedVer:    "1.0.0",
			shouldError:    false,
		},
		{
			name:           "https://example.com:8080/path:myapp@v1.2.3",
			spec:           "https://example.com:8080/path:myapp@v1.2.3",
			expectedSource: "https://example.com:8080/path",
			expectedPkg:    "myapp",
			expectedVer:    "v1.2.3",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path:myapp@>=1.0.0",
			spec:           "https://example.com/path:myapp@>=1.0.0",
			expectedSource: "https://example.com/path",
			expectedPkg:    "myapp",
			expectedVer:    ">=1.0.0",
			shouldError:    false,
		},

		// Валидные случаи из документации: URL@version
		{
			name:           "http://example.com/path@1.0.0",
			spec:           "http://example.com/path@1.0.0",
			expectedSource: "http://example.com/path",
			expectedPkg:    "",
			expectedVer:    "1.0.0",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path@v1.2.3",
			spec:           "https://example.com/path@v1.2.3",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    "v1.2.3",
			shouldError:    false,
		},
		{
			name:           "https://github.com/owner/repo@^1.0.0",
			spec:           "https://github.com/owner/repo@^1.0.0",
			expectedSource: "https://github.com/owner/repo",
			expectedPkg:    "",
			expectedVer:    "^1.0.0",
			shouldError:    false,
		},

		// Валидные случаи: URL без версии и без имени пакета
		{
			name:           "file:///Users/alex/projects/seniorGolang/tg-proxy",
			spec:           "file:///Users/alex/projects/seniorGolang/tg-proxy",
			expectedSource: "file:///Users/alex/projects/seniorGolang/tg-proxy",
			expectedPkg:    "",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path",
			spec:           "https://example.com/path",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    "",
			shouldError:    false,
		},

		// Валидные случаи: URL с именем пакета без версии
		{
			name:           "file:///path/to/manifest.yml:mypackage",
			spec:           "file:///path/to/manifest.yml:mypackage",
			expectedSource: "file:///path/to/manifest.yml",
			expectedPkg:    "mypackage",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path:mypackage",
			spec:           "https://example.com/path:mypackage",
			expectedSource: "https://example.com/path",
			expectedPkg:    "mypackage",
			expectedVer:    "",
			shouldError:    false,
		},

		// Валидные случаи: URL с портом
		{
			name:           "https://example.com:8080/path",
			spec:           "https://example.com:8080/path",
			expectedSource: "https://example.com:8080/path",
			expectedPkg:    "",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "http://example.com:80/path",
			spec:           "http://example.com:80/path",
			expectedSource: "http://example.com:80/path",
			expectedPkg:    "",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "https://example.com:443/path",
			spec:           "https://example.com:443/path",
			expectedSource: "https://example.com:443/path",
			expectedPkg:    "",
			expectedVer:    "",
			shouldError:    false,
		},

		// Валидные случаи: URL с портом и именем пакета
		{
			name:           "https://example.com:8080/path:mypackage",
			spec:           "https://example.com:8080/path:mypackage",
			expectedSource: "https://example.com:8080/path",
			expectedPkg:    "mypackage",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "https://example.com:8080/path:mypackage@1.0.0",
			spec:           "https://example.com:8080/path:mypackage@1.0.0",
			expectedSource: "https://example.com:8080/path",
			expectedPkg:    "mypackage",
			expectedVer:    "1.0.0",
			shouldError:    false,
		},

		// Валидные случаи: ограничения версии
		{
			name:           "https://example.com/path@>=1.0.0",
			spec:           "https://example.com/path@>=1.0.0",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    ">=1.0.0",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path@<=1.0.0",
			spec:           "https://example.com/path@<=1.0.0",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    "<=1.0.0",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path@>1.0.0",
			spec:           "https://example.com/path@>1.0.0",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    ">1.0.0",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path@<1.0.0",
			spec:           "https://example.com/path@<1.0.0",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    "<1.0.0",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path@~1.0.0",
			spec:           "https://example.com/path@~1.0.0",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    "~1.0.0",
			shouldError:    false,
		},
		{
			name:           "https://example.com/path@^1.0.0",
			spec:           "https://example.com/path@^1.0.0",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    "^1.0.0",
			shouldError:    false,
		},

		// Пограничные валидные случаи
		{
			name:           "file://",
			spec:           "file://",
			expectedSource: "file://",
			expectedPkg:    "",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "file:///",
			spec:           "file:///",
			expectedSource: "file:///",
			expectedPkg:    "",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "file://:mypackage",
			spec:           "file://:mypackage",
			expectedSource: "file://",
			expectedPkg:    "mypackage",
			expectedVer:    "",
			shouldError:    false,
		},
		{
			name:           "URL с пробелами в версии (обрезаются)",
			spec:           "https://example.com/path@ 1.0.0 ",
			expectedSource: "https://example.com/path",
			expectedPkg:    "",
			expectedVer:    "1.0.0",
			shouldError:    false,
		},

		// Ошибки: нет схемы
		{
			name:          "URL без схемы",
			spec:          "/path/to/file",
			shouldError:   true,
			errorContains: "URL must have a scheme",
		},
		{
			name:          "Пустая строка",
			spec:          "",
			shouldError:   true,
			errorContains: "URL must have a scheme",
		},
		{
			name:          "Только имя пакета",
			spec:          "mypackage",
			shouldError:   true,
			errorContains: "URL must have a scheme",
		},
		{
			name:          "Только имя пакета с версией",
			spec:          "mypackage@1.0.0",
			shouldError:   true,
			errorContains: "URL must have a scheme",
		},
		{
			name:          "URL с неполной схемой (только : без //)",
			spec:          "file:/path",
			shouldError:   true,
			errorContains: "URL must have a scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &URI{}
			err := u.parse(tt.spec)

			if tt.shouldError {
				if err == nil {
					t.Errorf("parse() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("parse() error = %v, expected to contain %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parse() unexpected error = %v", err)
				return
			}

			if u.source != tt.expectedSource {
				t.Errorf("parse() source = %q, want %q", u.source, tt.expectedSource)
			}

			if u.packageName != tt.expectedPkg {
				t.Errorf("parse() packageName = %q, want %q", u.packageName, tt.expectedPkg)
			}

			if u.version.Original != tt.expectedVer {
				t.Errorf("parse() version.Original = %q, want %q", u.version.Original, tt.expectedVer)
			}

			// Проверяем, что parsedURL установлен
			if u.parsedURL == nil {
				t.Errorf("parse() parsedURL is nil")
			} else if u.parsedURL.Scheme == "" {
				t.Errorf("parse() parsedURL.Scheme is empty")
			}
		})
	}
}

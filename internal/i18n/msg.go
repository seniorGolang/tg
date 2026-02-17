// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package i18n

func Msg(text string) (translated string) {

	if translation, ok := translations[text]; ok {
		return translation
	}
	return text
}

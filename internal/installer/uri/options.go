// Copyright (c) 2025 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package uri

type Option func(u *URI)

func Dist(d ...dist) (opt Option) {

	return func(u *URI) {
		u.dists = append(u.dists, d...)
	}
}

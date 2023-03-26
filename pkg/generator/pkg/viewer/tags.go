package viewer

import (
	"strings"
)

const tagName = "dumper"

func tagToOption(tag string) (opt option) {

	parsed := strings.Split(tag, ",")
	if len(parsed) == 2 {
		switch parsed[0] {
		case "hide":
			return hide(parsed[1])
		}
	}
	return
}

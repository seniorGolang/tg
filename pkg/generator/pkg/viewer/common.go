package viewer

import (
	"io"
	"strconv"
)

var (
	plusBytes       = []byte("+")
	iBytes          = []byte("i")
	trueBytes       = []byte("true")
	falseBytes      = []byte("false")
	interfaceBytes  = []byte("(interface {})")
	openBraceBytes  = []byte("{")
	closeBraceBytes = []byte("}")
	asteriskBytes   = []byte("*")
	colonBytes      = []byte(":")
	openParenBytes  = []byte("(")
	closeParenBytes = []byte(")")
	spaceBytes      = []byte(" ")
	// pointerChainBytes  = []byte("->")
	nilAngleBytes      = []byte("<nil>")
	maxShortBytes      = []byte("<max>")
	circularShortBytes = []byte("<shown>")
	invalidAngleBytes  = []byte("<invalid>")
	openBracketBytes   = []byte("[")
	closeBracketBytes  = []byte("]")
	percentBytes       = []byte("%")
	precisionBytes     = []byte(".")
	// openAngleBytes     = []byte("<")
	// closeAngleBytes    = []byte(">")
	openMapBytes  = []byte("map[")
	closeMapBytes = []byte("]")
)

var hexDigits = "0123456789abcdef"

func printBool(w io.Writer, val bool) {
	if val {
		_, _ = w.Write(trueBytes)
	} else {
		_, _ = w.Write(falseBytes)
	}
}

func intBytes(val int64, base int) []byte {
	return []byte(strconv.FormatInt(val, base))
}

func uintBytes(val uint64, base int) []byte {
	return []byte(strconv.FormatUint(val, base))
}

func floatBytes(val float64, precision int) []byte {
	return []byte(strconv.FormatFloat(val, 'g', -1, precision))
}

func printComplex(w io.Writer, c complex128, floatPrecision int) {
	r := real(c)
	_, _ = w.Write(openParenBytes)
	_, _ = w.Write([]byte(strconv.FormatFloat(r, 'g', -1, floatPrecision)))
	i := imag(c)
	if i >= 0 {
		_, _ = w.Write(plusBytes)
	}
	_, _ = w.Write([]byte(strconv.FormatFloat(i, 'g', -1, floatPrecision)))
	_, _ = w.Write(iBytes)
	_, _ = w.Write(closeParenBytes)
}

func printHexPtr(w io.Writer, p uintptr) {

	num := uint64(p)
	if num == 0 {
		_, _ = w.Write(nilAngleBytes)
		return
	}

	buf := make([]byte, 18)

	base := uint64(16)
	i := len(buf) - 1
	for num >= base {
		buf[i] = hexDigits[num%base]
		num /= base
		i--
	}
	buf[i] = hexDigits[num]

	i--
	buf[i] = 'x'
	i--
	buf[i] = '0'

	buf = buf[i:]
	_, _ = w.Write(buf)
}

// Copyright (c) 2020 Khramtsov Aleksei (contact@altsoftllc.com).
// This file (swagger-utils.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

func codeToText(code int) string {
	if text, found := statusText[code]; found {
		return text
	}
	return fmt.Sprintf("unknown error %d", code)
}

var statusText = map[int]string{
	fasthttp.StatusContinue:                      "Continue",
	fasthttp.StatusSwitchingProtocols:            "Switching Protocols",
	fasthttp.StatusProcessing:                    "Processing",
	fasthttp.StatusOK:                            "Successful operation",
	fasthttp.StatusCreated:                       "Created",
	fasthttp.StatusAccepted:                      "Accepted",
	fasthttp.StatusNonAuthoritativeInfo:          "Non-Authoritative Information",
	fasthttp.StatusNoContent:                     "No Content",
	fasthttp.StatusResetContent:                  "Reset Content",
	fasthttp.StatusPartialContent:                "Partial Content",
	fasthttp.StatusMultiStatus:                   "Multi-Status",
	fasthttp.StatusAlreadyReported:               "Already Reported",
	fasthttp.StatusIMUsed:                        "IM Used",
	fasthttp.StatusMultipleChoices:               "Multiple Choices",
	fasthttp.StatusMovedPermanently:              "Moved Permanently",
	fasthttp.StatusFound:                         "Found",
	fasthttp.StatusSeeOther:                      "See Other",
	fasthttp.StatusNotModified:                   "Not Modified",
	fasthttp.StatusUseProxy:                      "Use Proxy",
	fasthttp.StatusTemporaryRedirect:             "Temporary Redirect",
	fasthttp.StatusPermanentRedirect:             "Permanent Redirect",
	fasthttp.StatusBadRequest:                    "Bad Request",
	fasthttp.StatusUnauthorized:                  "Unauthorized",
	fasthttp.StatusPaymentRequired:               "Payment Required",
	fasthttp.StatusForbidden:                     "Forbidden",
	fasthttp.StatusNotFound:                      "Not Found",
	fasthttp.StatusMethodNotAllowed:              "Method Not Allowed",
	fasthttp.StatusNotAcceptable:                 "Not Acceptable",
	fasthttp.StatusProxyAuthRequired:             "Proxy Authentication Required",
	fasthttp.StatusRequestTimeout:                "Request Timeout",
	fasthttp.StatusConflict:                      "Conflict",
	fasthttp.StatusGone:                          "Gone",
	fasthttp.StatusLengthRequired:                "Length Required",
	fasthttp.StatusPreconditionFailed:            "Precondition Failed",
	fasthttp.StatusRequestEntityTooLarge:         "Request Entity Too Large",
	fasthttp.StatusRequestURITooLong:             "Request URI Too Long",
	fasthttp.StatusUnsupportedMediaType:          "Unsupported Media Type",
	fasthttp.StatusRequestedRangeNotSatisfiable:  "Requested Range Not Satisfiable",
	fasthttp.StatusExpectationFailed:             "Expectation Failed",
	fasthttp.StatusTeapot:                        "I'm a teapot",
	fasthttp.StatusUnprocessableEntity:           "Unprocessable Entity",
	fasthttp.StatusLocked:                        "Locked",
	fasthttp.StatusFailedDependency:              "Failed Dependency",
	fasthttp.StatusUpgradeRequired:               "Upgrade Required",
	fasthttp.StatusPreconditionRequired:          "Precondition Required",
	fasthttp.StatusTooManyRequests:               "Too Many Requests",
	fasthttp.StatusRequestHeaderFieldsTooLarge:   "Request Header Fields Too Large",
	fasthttp.StatusUnavailableForLegalReasons:    "Unavailable For Legal Reasons",
	fasthttp.StatusInternalServerError:           "Internal Server Error",
	fasthttp.StatusNotImplemented:                "Not Implemented",
	fasthttp.StatusBadGateway:                    "Bad Gateway",
	fasthttp.StatusServiceUnavailable:            "Service Unavailable",
	fasthttp.StatusGatewayTimeout:                "Gateway Timeout",
	fasthttp.StatusHTTPVersionNotSupported:       "HTTP Version Not Supported",
	fasthttp.StatusVariantAlsoNegotiates:         "Variant Also Negotiates",
	fasthttp.StatusInsufficientStorage:           "Insufficient Storage",
	fasthttp.StatusLoopDetected:                  "Loop Detected",
	fasthttp.StatusNotExtended:                   "Not Extended",
	fasthttp.StatusNetworkAuthenticationRequired: "Network Authentication Required",
}

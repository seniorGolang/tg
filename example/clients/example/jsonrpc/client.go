package jsonrpc

import (
	"crypto/tls"
	"net/http"

	"github.com/google/uuid"
)

const (
	Version = "2.0"
)

type ID = uuid.UUID

var NilID = uuid.Nil
var NewID = uuid.New

type ClientRPC struct {
	options    options
	endpoint   string
	httpClient *http.Client
}

func NewClient(endpoint string, opts ...Option) (client *ClientRPC) {

	client = &ClientRPC{
		endpoint:   endpoint,
		httpClient: &http.Client{},
		options:    prepareOpts(opts),
	}
	if client.options.insecure {
		client.httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return client
}

package jsonrpc

import (
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
		httpClient: http.DefaultClient,
		options:    prepareOpts(opts),
	}
	if client.options.clientHTTP != nil {
		client.httpClient = client.options.clientHTTP
	}
	if client.options.tlsConfig != nil {
		client.httpClient.Transport = &http.Transport{
			TLSClientConfig: client.options.tlsConfig,
		}
	}
	return client
}

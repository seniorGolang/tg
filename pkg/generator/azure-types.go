package generator

type azureFunc struct {
	Bindings []azureBindings `json:"bindings"`
}

type azureBindings struct {
	Name      string   `json:"name,omitempty"`
	Type      string   `json:"type,omitempty"`
	Route     string   `json:"route,omitempty"`
	Methods   []string `json:"methods,omitempty"`
	AuthLevel string   `json:"authLevel,omitempty"`
	Direction string   `json:"direction,omitempty"`
}

type azureHost struct {
	Version         string                `json:"version"`
	ExtensionBundle *azureExtensionBundle `json:"extensionBundle,omitempty"`
	Aggregator      *azureAggregator      `json:"aggregator,omitempty"`
	Extensions      *azureExtensions      `json:"extensions,omitempty"`
	FunctionTimeout string                `json:"functionTimeout,omitempty"`
	HealthMonitor   *azureHealthMonitor   `json:"healthMonitor,omitempty"`
	Logging         *azureLogging         `json:"logging,omitempty"`
	CustomHandler   *azureCustomHandler   `json:"customHandler,omitempty"`
}

type azureAggregator struct {
	BatchSize    int64  `json:"batchSize,omitempty"`
	FlushTimeout string `json:"flushTimeout,omitempty"`
}

type azureCustomHandler struct {
	Description                 *azureDescription `json:"description,omitempty"`
	EnableForwardingHTTPRequest bool              `json:"enableForwardingHttpRequest,omitempty"`
}

type azureDescription struct {
	DefaultExecutablePath string `json:"defaultExecutablePath,omitempty"`
}

type azureExtensionBundle struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

type azureExtensions struct {
	HTTP *azureHTTP `json:"http,omitempty"`
}

type azureHTTP struct {
	RoutePrefix             string     `json:"routePrefix"`
	MaxOutstandingRequests  int64      `json:"maxOutstandingRequests,omitempty"`
	MaxConcurrentRequests   int64      `json:"maxConcurrentRequests,omitempty"`
	DynamicThrottlesEnabled bool       `json:"dynamicThrottlesEnabled,omitempty"`
	Hsts                    *azureHsts `json:"hsts,omitempty"`
}

type azureHsts struct {
	IsEnabled bool   `json:"isEnabled,omitempty"`
	MaxAge    string `json:"maxAge,omitempty"`
}

type azureHealthMonitor struct {
	Enabled              bool    `json:"enabled,omitempty"`
	HealthCheckInterval  string  `json:"healthCheckInterval,omitempty"`
	HealthCheckWindow    string  `json:"healthCheckWindow,omitempty"`
	HealthCheckThreshold int64   `json:"healthCheckThreshold,omitempty"`
	CounterThreshold     float64 `json:"counterThreshold,omitempty"`
}

type azureLogging struct {
	FileLoggingMode string         `json:"fileLoggingMode,omitempty"`
	LogLevel        *azureLogLevel `json:"logLevel,omitempty"`
}

type azureLogLevel struct {
	Default string `json:"default,omitempty"`
}

package generator

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

type azure struct {
	*Transport
}

func newAzure(tr *Transport) (doc *azure) {

	doc = &azure{Transport: tr}
	return
}

func (app *azure) render(appName, routePrefix, outFilePath, logLevel string, enableHealth bool) (err error) {

	outFilePath, _ = filepath.Abs(outFilePath)

	for _, svcName := range app.serviceKeys() {
		svc := app.services[svcName]
		for _, svcMethod := range svc.methods {
			route := svcMethod.httpPath(false)
			if svcMethod.isJsonRPC() {
				route = svcMethod.jsonrpcPath(false)
			}
			fn := azureFunc{
				Bindings: []azureBindings{
					{
						Name:      "req",
						Direction: "in",
						AuthLevel: "anonymous",
						Type:      "httpTrigger",
						Route:     route,
						Methods:   []string{"head", "options", svcMethod.httpMethod()},
					},
					{
						Name:      "res",
						Type:      "http",
						Direction: "out",
					},
				},
			}
			outFileName := path.Join(outFilePath, appName, utils.ToLowerCamel(svcName)+svcMethod.Name, "function.json")
			if err = os.MkdirAll(filepath.Dir(outFileName), 0777); err != nil {
				return
			}
			if err = os.WriteFile(outFileName, toJSON(fn), 0600); err != nil {
				return
			}
		}
	}
	host := azureHost{
		Version: "2.0",
		ExtensionBundle: &azureExtensionBundle{
			ID:      "Microsoft.Azure.Functions.ExtensionBundle",
			Version: "[2.*,3.0.0)",
		},
		Extensions: &azureExtensions{
			HTTP: &azureHTTP{
				RoutePrefix: routePrefix,
			},
		},
		HealthMonitor:   &azureHealthMonitor{},
		Aggregator:      &azureAggregator{},
		FunctionTimeout: "",
		Logging: &azureLogging{
			FileLoggingMode: "debugOnly",
			LogLevel:        &azureLogLevel{Default: logLevel},
		},
		CustomHandler: &azureCustomHandler{
			Description:                 &azureDescription{DefaultExecutablePath: "runner"},
			EnableForwardingHTTPRequest: true,
		},
	}
	if enableHealth {
		host.HealthMonitor = &azureHealthMonitor{
			Enabled:              enableHealth,
			HealthCheckInterval:  "00:00:10",
			HealthCheckWindow:    "00:02:00",
			HealthCheckThreshold: 6,
			CounterThreshold:     0.80,
		}
	}
	outFileName := path.Join(outFilePath, appName, "host.json")
	if _, fsErr := os.Stat(outFileName); os.IsNotExist(fsErr) {
		if err = os.MkdirAll(filepath.Dir(outFileName), 0777); err != nil {
			return
		}
		return os.WriteFile(outFileName, toJSON(host), 0600)
	}
	return
}

func toJSON(v interface{}) (data []byte) {
	data, _ = json.MarshalIndent(v, "", " ")
	return
}

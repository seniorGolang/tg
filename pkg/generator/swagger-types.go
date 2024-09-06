// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (swagger-types.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

type swObject struct {
	OpenAPI    string            `json:"openapi" yaml:"openapi"`
	Info       swInfo            `json:"info,omitempty" yaml:"info,omitempty"`
	Servers    []swServer        `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []swTag           `json:"tags,omitempty" yaml:"tags,omitempty"`
	Schemes    []string          `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	Paths      map[string]swPath `json:"paths" yaml:"paths"`
	Components swComponents      `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []swSecurity      `json:"security,omitempty" yaml:"security,omitempty"`
}

type swPath struct {
	Ref         string       `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary     string       `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	Get         *swOperation `json:"get,omitempty" yaml:"get,omitempty"`
	Post        *swOperation `json:"post,omitempty" yaml:"post,omitempty"`
	Patch       *swOperation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Put         *swOperation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete      *swOperation `json:"delete,omitempty" yaml:"delete,omitempty"`
}

type swOperation struct {
	Tags        []string       `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string         `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Consumes    []string       `json:"consumes,omitempty" yaml:"consumes,omitempty"`
	Produces    []string       `json:"produces,omitempty" yaml:"produces,omitempty"`
	Parameters  []swParameter  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *swRequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   swResponses    `json:"responses,omitempty" yaml:"responses,omitempty"`
	Deprecated  bool           `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Servers     []swServer     `json:"servers,omitempty" yaml:"servers,omitempty"`
	CodeSamples []swCodeSample `json:"x-code-samples,omitempty" yaml:"x-code-samples,omitempty"`
}

type swCodeSample struct {
	Lang   string `json:"lang" yaml:"lang"`
	Source string `json:"source" yaml:"source"`
}

type swContact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type swLicense struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

type swInfo struct {
	Title          string     `json:"title,omitempty" yaml:"title,omitempty"`
	Description    string     `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string     `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *swContact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *swLicense `json:"license,omitempty" yaml:"license,omitempty"`
	Version        string     `json:"version,omitempty" yaml:"version,omitempty"`
}

type swExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url,omitempty" yaml:"url,omitempty"`
}

type swTag struct {
	Name         string         `json:"name,omitempty" yaml:"name,omitempty"`
	Description  string         `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs swExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

type swServer struct {
	URL         string                `json:"url,omitempty" yaml:"url,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]swVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

type swSchemas map[string]swSchema

type swProperties map[string]swSchema

type swComponents struct {
	Schemas         swSchemas          `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes *swSecuritySchemes `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type swSchema struct {
	Ref         string       `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type        string       `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string       `json:"format,omitempty" yaml:"format,omitempty"`
	Minimum     int          `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum     int          `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Required    []string     `json:"required,omitempty" yaml:"required,omitempty"`
	Properties  swProperties `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items       *swSchema    `json:"items,omitempty" yaml:"items,omitempty"`
	Enum        []string     `json:"enum,omitempty" yaml:"enum,omitempty"`
	Nullable    bool         `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Example     interface{}  `json:"example,omitempty" yaml:"example,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`

	OneOf []swSchema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AllOf []swSchema `json:"allOf,omitempty" yaml:"allOf,omitempty"`

	AdditionalProperties interface{} `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
}

type swVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

type swParameter struct {
	Ref         string   `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	In          string   `json:"in,omitempty" yaml:"in,omitempty"`
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool     `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      swSchema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type swMedia struct {
	Schema swSchema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type swContent map[string]swMedia

type swResponse struct {
	Description string              `json:"description" yaml:"description"`
	Content     swContent           `json:"content,omitempty" yaml:"content,omitempty"`
	Headers     map[string]swHeader `json:"headers,omitempty" yaml:"headers,omitempty"`
}

type swHeader struct {
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      swSchema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type swResponses map[string]swResponse

type swRequestBody struct {
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Content     swContent `json:"content,omitempty" yaml:"content,omitempty"`
}

type swSecurity struct {
	BearerAuth []interface{} `json:"bearerAuth" yaml:"bearerAuth"`
}

type swSecuritySchemes struct {
	BearerAuth swBearerAuth `json:"bearerAuth,omitempty" yaml:"bearerAuth,omitempty"`
}

type swBearerAuth struct {
	Type   string `json:"type,omitempty" yaml:"type,omitempty"`
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (swagger-jsonrpc.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

func jsonrpcSchema(propName string, property swSchema) (schema swSchema) {

	schema = swSchema{
		Type:     "object",
		Required: []string{"id", "jsonrpc", "result"},
		Properties: swProperties{
			"id": swSchema{
				Example: 1,
				OneOf:   []swSchema{{Type: "number"}, {Type: "string", Format: "uuid"}},
			},
			"jsonrpc": swSchema{
				Type:    "string",
				Example: "2.0",
			},
		},
	}

	schema.Properties[propName] = property
	return
}

func jsonrpcErrorSchema() (schema swSchema) {

	schema = swSchema{
		Type:     "object",
		Required: []string{"id", "jsonrpc", "error"},
		Properties: swProperties{
			"id": swSchema{
				Example: 1,
				OneOf:   []swSchema{{Type: "number"}, {Type: "string", Format: "uuid"}},
			},
			"jsonrpc": swSchema{
				Type:    "string",
				Example: "2.0",
			},
			"error": swSchema{
				Type: "object",
				Properties: swProperties{
					"code": swSchema{
						Example: -32603,
						Type:    "number",
						Format:  "int32",
					},
					"message": swSchema{
						Type:    "string",
						Example: "not found",
					},
					"data": swSchema{
						Type:     "object",
						Nullable: true,
					},
				},
			},
		},
	}
	return
}

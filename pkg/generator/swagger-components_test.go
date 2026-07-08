package generator

import (
	"reflect"
	"testing"

	"github.com/seniorGolang/tg/v2/pkg/astra/types"
)

func TestWalkVariableMarksStructFieldsRequiredByJSONPresence(t *testing.T) {
	doc := &swagger{
		schemas:    make(swSchemas),
		knownCount: make(map[string]int),
		knownTypes: make(map[string]swSchema),
	}

	stringType := types.TName{TypeName: "string"}
	schema := doc.walkVariable("Sample", "example/service", types.Struct{
		Base: types.Base{Name: "Sample"},
		Fields: []types.StructField{
			{
				Variable: types.Variable{Base: types.Base{Name: "Required"}, Type: stringType},
				Tags:     map[string][]string{"json": {"required"}},
			},
			{
				Variable: types.Variable{Base: types.Base{Name: "RequiredWithoutJSONTag"}, Type: stringType},
			},
			{
				Variable: types.Variable{Base: types.Base{Name: "OptionalByOmitEmpty"}, Type: stringType},
				Tags:     map[string][]string{"json": {"optionalByOmitEmpty", "omitempty"}},
			},
			{
				Variable: types.Variable{Base: types.Base{Name: "OptionalWithDefaultJSONName"}, Type: stringType},
				Tags:     map[string][]string{"json": {"", "omitempty"}},
			},
			{
				Variable: types.Variable{
					Base: types.Base{Name: "OptionalPointer"},
					Type: types.TPointer{NumberOfPointers: 1, Next: stringType},
				},
				Tags: map[string][]string{"json": {"optionalPointer"}},
			},
			{
				Variable: types.Variable{
					Base: types.Base{
						Name: "ForcedPointer",
						Docs: []string{"// @tg required"},
					},
					Type: types.TPointer{NumberOfPointers: 1, Next: stringType},
				},
				Tags: map[string][]string{"json": {"forcedPointer"}},
			},
			{
				Variable: types.Variable{Base: types.Base{Name: "Ignored"}, Type: stringType},
				Tags:     map[string][]string{"json": {"-"}},
			},
			{
				Variable: types.Variable{Base: types.Base{Name: "hidden"}, Type: stringType},
				Tags:     map[string][]string{"json": {"hidden"}},
			},
		},
	}, nil)

	want := []string{"required", "RequiredWithoutJSONTag", "optionalPointer", "forcedPointer"}
	if !reflect.DeepEqual(schema.Required, want) {
		t.Fatalf("required fields = %v, want %v", schema.Required, want)
	}
	if _, found := schema.Properties["OptionalWithDefaultJSONName"]; !found {
		t.Fatal("expected field with empty json tag name to use Go field name")
	}
	if _, found := schema.Properties[""]; found {
		t.Fatal("unexpected empty property name")
	}
}

func TestRegisterStructKeepsPointerFieldsOptional(t *testing.T) {
	doc := &swagger{
		schemas:    make(swSchemas),
		knownCount: make(map[string]int),
		knownTypes: make(map[string]swSchema),
	}

	stringType := types.TName{TypeName: "string"}
	doc.registerStruct("requestSample", "example/service", nil, []types.StructField{
		{
			Variable: types.Variable{Base: types.Base{Name: "Required"}, Type: stringType},
			Tags:     map[string][]string{"json": {"required"}},
		},
		{
			Variable: types.Variable{
				Base: types.Base{Name: "OptionalPointer"},
				Type: types.TPointer{NumberOfPointers: 1, Next: stringType},
			},
			Tags: map[string][]string{"json": {"optionalPointer"}},
		},
		{
			Variable: types.Variable{Base: types.Base{Name: "OptionalByOmitEmpty"}, Type: stringType},
			Tags:     map[string][]string{"json": {"optionalByOmitEmpty", "omitempty"}},
		},
	}, isRequiredGeneratedRequestField)

	want := []string{"required"}
	if !reflect.DeepEqual(doc.schemas["requestSample"].Required, want) {
		t.Fatalf("required fields = %v, want %v", doc.schemas["requestSample"].Required, want)
	}
}

func TestRegisterStructMarksResponsePointerFieldsRequired(t *testing.T) {
	doc := &swagger{
		schemas:    make(swSchemas),
		knownCount: make(map[string]int),
		knownTypes: make(map[string]swSchema),
	}

	stringType := types.TName{TypeName: "string"}
	doc.registerStruct("responseSample", "example/service", nil, []types.StructField{
		{
			Variable: types.Variable{Base: types.Base{Name: "TotalCount"}, Type: types.TName{TypeName: "int"}},
			Tags:     map[string][]string{"json": {"totalCount"}},
		},
		{
			Variable: types.Variable{
				Base: types.Base{Name: "Response"},
				Type: types.TPointer{NumberOfPointers: 1, Next: stringType},
			},
			Tags: map[string][]string{"json": {"response"}},
		},
	}, isRequiredGeneratedResponseField)

	want := []string{"totalCount", "response"}
	if !reflect.DeepEqual(doc.schemas["responseSample"].Required, want) {
		t.Fatalf("required fields = %v, want %v", doc.schemas["responseSample"].Required, want)
	}
}

func TestWalkVariableKeepsUUIDPointersNullable(t *testing.T) {
	doc := &swagger{
		schemas:    make(swSchemas),
		knownCount: make(map[string]int),
		knownTypes: make(map[string]swSchema),
	}

	schema := doc.walkVariable("DraftID", "example/service", types.TPointer{
		NumberOfPointers: 1,
		Next: types.TImport{
			Import: &types.Import{
				Base:    types.Base{Name: "uuid"},
				Package: "github.com/google/uuid",
			},
			Next: types.TName{TypeName: "UUID"},
		},
	}, nil)

	want := []swSchema{
		{Type: "string", Format: "uuid"},
		{Nullable: true},
	}
	if !reflect.DeepEqual(schema.OneOf, want) {
		t.Fatalf("oneOf = %#v, want %#v", schema.OneOf, want)
	}
}

func TestWalkVariableKeepsKnownSchemaPointersNullable(t *testing.T) {
	doc := &swagger{
		schemas: swSchemas{
			"service.Known": {Type: "object"},
		},
		knownCount: make(map[string]int),
		knownTypes: make(map[string]swSchema),
	}

	schema := doc.walkVariable("Known", "example/service", types.TPointer{
		NumberOfPointers: 1,
		Next:             types.TName{TypeName: "Known"},
	}, nil)

	want := []swSchema{
		{Ref: "#/components/schemas/service.Known"},
		{Nullable: true},
	}
	if !reflect.DeepEqual(schema.OneOf, want) {
		t.Fatalf("oneOf = %#v, want %#v", schema.OneOf, want)
	}
}

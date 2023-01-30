package jsonrpc

import (
	"reflect"
)

func Params(params ...interface{}) interface{} {

	var finalParams interface{}
	if params != nil {
		switch len(params) {
		case 0:
		case 1:
			if params[0] != nil {
				var typeOf reflect.Type
				for typeOf = reflect.TypeOf(params[0]); typeOf != nil && typeOf.Kind() == reflect.Ptr; typeOf = typeOf.Elem() {
				}
				if typeOf != nil {
					switch typeOf.Kind() {
					case reflect.Struct:
						finalParams = params[0]
					case reflect.Array:
						finalParams = params[0]
					case reflect.Slice:
						finalParams = params[0]
					case reflect.Interface:
						finalParams = params[0]
					case reflect.Map:
						finalParams = params[0]
					default:
						finalParams = params
					}
				}
			} else {
				finalParams = params
			}
		default:
			finalParams = params
		}
	}
	return finalParams
}

package handler

import (
	"reflect"
	"strings"

	"github.com/zdnscloud/cement/reflector"
	"github.com/zdnscloud/gorest/types"
)

func CheckObjectFields(ctx *types.Context) *types.APIError {
	structVal, ok := reflector.GetStructFromPointer(ctx.Object)
	if ok == false {
		return types.NewAPIError(types.ServerError, "get object structure but return "+structVal.Kind().String())
	}

	_, err := getStructValue(ctx, ctx.Object.GetSchema(), structVal)
	return err
}

func getStructValue(ctx *types.Context, schema *types.Schema, structVal reflect.Value) (map[string]interface{}, *types.APIError) {
	fieldValues := map[string]interface{}{}
	structTyp := structVal.Type()
	if schema == nil {
		schema = ctx.Schemas.Schema(&ctx.Object.GetSchema().Version, strings.ToLower(structTyp.Name()))
		if schema == nil {
			return nil, types.NewAPIError(types.NotFound, "no found schema "+strings.ToLower(structTyp.Name()))
		}
	}

	for i := 0; i < structVal.NumField(); i++ {
		field := structTyp.Field(i)
		fieldJsonName, isAnonymous := types.GetFieldJsonName(field)
		fieldVal := structVal.FieldByName(field.Name)
		if fieldVal.IsValid() == false {
			continue
		}

		if isAnonymous {
			if _, err := getStructValue(ctx, ctx.Object.GetSchema(), fieldVal); err != nil {
				return nil, err
			}
			continue
		}

		if fieldJsonName == "" {
			continue
		}

		value, err := getFieldValue(ctx, fieldVal)
		if err != nil {
			return nil, err
		}

		schemaField := schema.ResourceFields[fieldJsonName]
		if valueIsNil(value) && schemaField.Required {
			return nil, types.NewAPIError(types.MissingRequired, "field "+fieldJsonName+" must be set when create")
		}

		fieldValues[fieldJsonName] = value
	}

	return fieldValues, nil
}

func valueIsNil(value interface{}) bool {
	if value == nil || value == "" || value == 0 {
		return true
	}

	val := reflect.ValueOf(value)
	typ := val.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Map, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}

func getFieldValue(ctx *types.Context, fieldVal reflect.Value) (interface{}, *types.APIError) {
	fieldType := fieldVal.Type()
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
		fieldVal = reflect.Indirect(fieldVal)
	}

	if fieldVal.IsValid() == false {
		return nil, nil
	}

	switch fieldType.Kind() {
	case reflect.Struct:
		return getStructValue(ctx, nil, fieldVal)
	case reflect.Slice:
		return getSliceValue(ctx, fieldVal)
	case reflect.Map:
		return getMapValue(ctx, fieldVal)
	default:
		return fieldVal.Interface(), nil
	}
}

func getSliceValue(ctx *types.Context, fieldValSlice reflect.Value) (interface{}, *types.APIError) {
	var values []interface{}
	for i := 0; i < fieldValSlice.Len(); i++ {
		fieldVal := fieldValSlice.Index(i)
		if val, err := getFieldValue(ctx, fieldVal); err != nil {
			return nil, err
		} else {
			values = append(values, val)
		}
	}

	return values, nil
}

func getMapValue(ctx *types.Context, fieldValMap reflect.Value) (interface{}, *types.APIError) {
	values := map[string]interface{}{}
	for _, key := range fieldValMap.MapKeys() {
		val := fieldValMap.MapIndex(key)
		val = reflect.ValueOf(val.Interface())
		if fieldVal, err := getFieldValue(ctx, val); err != nil {
			return nil, err
		} else {
			values[key.String()] = fieldVal
		}
	}

	return values, nil
}

package resourcedoc

import (
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/util"
)

const (
	Array  = "array"
	Map    = "map"
	Enum   = "enum"
	Unknow = "unknow"
)

func cutSymbolUint(in string) string {
	switch in {
	case "int8", "int16", "int32", "int64":
		return string(util.Int)
	case "uint8", "uint16", "uint32", "uint64":
		return string(util.Uint)
	default:
		return in
	}
}

func setSlice(t reflect.Type) string {
	k := util.Inspect(t)
	switch k {
	case util.StringSlice:
		return string(util.String)
	case util.IntSlice, util.UintSlice, util.StructSlice, util.StructPtrSlice, util.BoolSlice:
		nestType := t.Elem()
		if k == util.StructPtrSlice {
			nestType = nestType.Elem()
		}
		return cutSymbolUint(nestType.Name())
	}
	return Unknow
}

func setMap(t reflect.Type) (string, string) {
	k := util.Inspect(t)
	switch k {
	case util.StringIntMap, util.StringStringMap, util.StringUintMap, util.StringStructMap, util.StringStructPtrMap:
		nestType := t.Elem()
		if k == util.StringStructPtrMap {
			nestType = nestType.Elem()
		}
		return string(util.String), cutSymbolUint(nestType.Name())
	}
	return Unknow, Unknow
}

func setType(t reflect.Type) string {
	k := util.Inspect(t)
	switch k {
	case util.String, util.Int, util.Uint, util.Bool:
		return string(k)
	case util.StringIntMap, util.StringStringMap, util.StringUintMap, util.StringStructMap, util.StringStructPtrMap:
		return Map
	case util.IntSlice, util.UintSlice, util.BoolSlice, util.StringSlice, util.StructSlice, util.StructPtrSlice:
		return Array
	case util.Struct:
		return t.Name()
	case util.StructPtr:
		return t.Elem().Name()
	}
	return Unknow
}

func LowerFirstCharacter(original string) string {
	if len(original) > 0 {
		original = strings.ToLower(original[:1]) + original[1:]
	}
	return original
}

func UpperFirstCharacter(original string) string {
	if len(original) > 0 {
		original = strings.ToUpper(original[:1]) + original[1:]
	}
	return original
}

func fieldJsonName(name, jsonTag string) string {
	if jsonTag != "" {
		tags := strings.Split(jsonTag, ",")
		for _, tag := range tags {
			if tag != "omitempty" {
				return tag
			}
		}
	}

	return name
}

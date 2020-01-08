package util

import (
	"fmt"
	"reflect"
)

type Kind string

const (
	Int    Kind = "int"
	Uint   Kind = "uint"
	Struct Kind = "struct"
	Bool   Kind = "bool"
	String Kind = "string"

	IntSlice       Kind = "intSlice"
	UintSlice      Kind = "uintSlice"
	StringSlice    Kind = "stringSlice"
	BoolSlice      Kind = "boolSlice"
	StructSlice    Kind = "structSlice"
	StructPtrSlice Kind = "structPtrSlice"

	StructPtr Kind = "structPtr"
	BoolPtr   Kind = "boolPtr"

	StringIntMap       Kind = "stringIntMap"
	StringUintMap      Kind = "stringUintMap"
	StringStringMap    Kind = "stringStringMap"
	StringStructMap    Kind = "stringStructMap"
	StringStructPtrMap Kind = "stringStructPtrMap"
)

func Inspect(typ reflect.Type) Kind {
	k := typ.Kind()
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Int
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Uint
	case reflect.String:
		return String
	case reflect.Bool:
		return Bool
	case reflect.Struct:
		return Struct
	case reflect.Ptr:
		vk := typ.Elem().Kind()
		if vk == reflect.Struct {
			return StructPtr
		}
		if vk == reflect.Bool {
			return BoolPtr
		}
	case reflect.Map:
		if typ.Key().Kind() == reflect.String {
			vk := typ.Elem().Kind()
			if vk == reflect.Struct {
				return StringStructMap
			} else if vk == reflect.Ptr {
				vk = typ.Elem().Elem().Kind()
				if vk == reflect.Struct {
					return StringStructPtrMap
				}
			} else {
				vkk, isPrimary := PrimartyKind(vk)
				if isPrimary {
					switch vkk {
					case Int:
						return StringIntMap
					case Uint:
						return StringUintMap
					case String:
						return StringStringMap
					}
				}
			}
		}
		return Kind(fmt.Sprintf("%s%sMap", typ.Name(), typ.Elem().Name()))
	case reflect.Slice:
		ek := typ.Elem().Kind()
		if ek == reflect.Struct {
			return StructSlice
		} else if ek == reflect.Ptr {
			ek = typ.Elem().Elem().Kind()
			if ek == reflect.Struct {
				return StructPtrSlice
			}
		} else {
			ekk, isPrimary := PrimartyKind(ek)
			if isPrimary {
				switch ekk {
				case Int:
					return IntSlice
				case Uint:
					return UintSlice
				case String:
					return StringSlice
				}
			}
		}
		return Kind(fmt.Sprintf("%sSlice", ek))
	}
	return Kind(fmt.Sprintf("Unkown(%s)", k))
}

func PrimartyKind(k reflect.Kind) (Kind, bool) {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Int, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Uint, true
	case reflect.String:
		return String, true
	case reflect.Bool:
		return Bool, true
	default:
		return "", false
	}
}

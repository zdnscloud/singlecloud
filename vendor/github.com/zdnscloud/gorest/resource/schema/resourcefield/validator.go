package resourcefield

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/zdnscloud/cement/slice"
)

type Validator interface {
	//validate each field is valid
	Validate(interface{}) error
}

type optionValidator struct {
	options []string
}

func newOptionValidator(options []string) Validator {
	return &optionValidator{options: options}
}

func (v *optionValidator) Validate(val interface{}) error {
	s, ok := val.(string)
	if ok == false {
		return fmt.Errorf("option can only used for string")
	}
	if slice.SliceIndex(v.options, s) == -1 {
		return fmt.Errorf("%s isn't included in options %v", s, v.options)
	}
	return nil

}

type stringLenRangeValidator struct {
	minLen int
	maxLen int
}

func newStringLenRangeValidator(minLen, maxLen int) Validator {
	return &stringLenRangeValidator{
		minLen: minLen,
		maxLen: maxLen,
	}
}

func (v *stringLenRangeValidator) Validate(val interface{}) error {
	s, ok := val.(string)
	if ok == false {
		return fmt.Errorf("string len range validator can only used for string")
	}
	l := len(s)
	if l < v.minLen || l >= v.maxLen {
		return fmt.Errorf("string len %d exceed the range limit[%v:%v)", l, v.minLen, v.maxLen)
	}
	return nil
}

type intRangeValidator struct {
	min int64
	max int64
}

func newIntRangeValidator(min, max int64) Validator {
	return &intRangeValidator{
		min: min,
		max: max,
	}
}

func (v *intRangeValidator) Validate(val interface{}) error {
	value := reflect.ValueOf(val)
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := value.Int()
		if i < v.min || i >= v.max {
			return fmt.Errorf("int value %v exceed the range limit[%v:%v)", i, v.min, v.max)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i := int64(value.Uint())
		if i < v.min || i >= v.max {
			return fmt.Errorf("uint value %v exceed the range limit[%v:%v)", i, v.min, v.max)
		}
	default:
		return fmt.Errorf("int range validator can only used for integers")
	}
	return nil
}

const (
	minTag           = "min="
	maxTag           = "max="
	minLenTag        = "minLen="
	maxLenTag        = "maxLen="
	optionsTag       = "options="
	optionsDelimiter = "|"
)

func buildValidator(fieldKind reflect.Kind, restTags []string) (Validator, error) {
	var minStr, maxStr string
	var minLenStr, maxLenStr string
	var options []string
	for _, tag := range restTags {
		if strings.HasPrefix(tag, minTag) {
			if minStr != "" {
				return nil, fmt.Errorf("has multi min tag")
			}
			minStr = strings.TrimPrefix(tag, minTag)
		} else if strings.HasPrefix(tag, maxTag) {
			if maxStr != "" {
				return nil, fmt.Errorf("has multi max tag")
			}
			maxStr = strings.TrimPrefix(tag, maxTag)
		} else if strings.HasPrefix(tag, optionsTag) {
			options = strings.Split(strings.TrimPrefix(tag, optionsTag), optionsDelimiter)
		} else if strings.HasPrefix(tag, minLenTag) {
			minLenStr = strings.TrimPrefix(tag, minLenTag)
		} else if strings.HasPrefix(tag, maxLenTag) {
			maxLenStr = strings.TrimPrefix(tag, maxLenTag)
		}
	}

	switch fieldKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if len(options) > 0 || minLenStr != "" || maxLenStr != "" {
			return nil, fmt.Errorf("integer field doesn't support options and min and max len")
		}

		if minStr == "" && maxStr == "" {
			return nil, nil
		} else if minStr != "" && maxStr == "" {
			return nil, fmt.Errorf("has min but not max")
		} else if minStr == "" && maxStr != "" {
			return nil, fmt.Errorf("has max but not min")
		} else {
			min, err := strconv.ParseInt(minStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("min value isn't valid int:%s", err.Error())
			}
			max, err := strconv.ParseInt(maxStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("max value isn't valid int:%s", err.Error())
			}
			if min >= max {
				return nil, fmt.Errorf("min value should smaller than max")
			}
			return newIntRangeValidator(min, max), nil
		}
	case reflect.String:
		if minStr != "" || maxStr != "" {
			return nil, fmt.Errorf("string field doesn't support options and min and max")
		}
		if len(options) != 0 {
			return newOptionValidator(options), nil
		} else if minLenStr == "" && maxLenStr == "" {
			return nil, nil
		} else if minLenStr == "" && maxLenStr != "" {
			return nil, fmt.Errorf("has maxLen but not minLen")
		} else if minLenStr != "" && maxLenStr == "" {
			return nil, fmt.Errorf("has minLen but not maxLen")
		} else {
			minLen, err := strconv.Atoi(minLenStr)
			if err != nil {
				return nil, fmt.Errorf("minLen value isn't valid int:%s", err.Error())
			}
			maxLen, err := strconv.Atoi(maxLenStr)
			if err != nil {
				return nil, fmt.Errorf("maxLen value isn't valid int:%s", err.Error())
			}
			if minLen >= maxLen {
				return nil, fmt.Errorf("min value should smaller than max")
			}
			return newStringLenRangeValidator(minLen, maxLen), nil
		}
	}

	return nil, nil
}

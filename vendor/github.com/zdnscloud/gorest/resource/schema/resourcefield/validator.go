package resourcefield

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gorest/util"
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
	value := reflect.ValueOf(val)
	kind := util.Inspect(value.Type())
	switch kind {
	case util.String:
		sv := value.String()
		if slice.SliceIndex(v.options, sv) == -1 {
			return fmt.Errorf("%s isn't included in options %v", sv, v.options)
		}
	case util.StringSlice:
		for i := 0; i < value.Len(); i++ {
			sv := value.Index(i).String()
			if slice.SliceIndex(v.options, sv) == -1 {
				return fmt.Errorf("%s isn't included in options %v", sv, v.options)
			}
		}
	}
	return nil

}

type domainNameValidator struct{}

var gDomainNameValidator Validator = &domainNameValidator{}

func newDomainNameValidator() Validator {
	return gDomainNameValidator
}

const (
	dns1123LabelFmt           string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123SubdomainFmt       string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"
	DNS1123SubdomainMaxLength int    = 253
)

var dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")

func (v *domainNameValidator) Validate(val interface{}) error {
	value := reflect.ValueOf(val)
	kind := util.Inspect(value.Type())
	switch kind {
	case util.String:
		sv := value.String()
		return validateDomain(sv)
	case util.StringSlice:
		for i := 0; i < value.Len(); i++ {
			sv := value.Index(i).String()
			if err := validateDomain(sv); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("domain name validator can only apply to string")
	}
}

func validateDomain(s string) error {
	if len(s) > DNS1123SubdomainMaxLength {
		return fmt.Errorf("exceed max domain name len limitation(253)")
	}

	if !dns1123SubdomainRegexp.MatchString(s) {
		return fmt.Errorf("subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character")
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
	value := reflect.ValueOf(val)
	kind := util.Inspect(value.Type())
	switch kind {
	case util.String:
		sv := value.String()
		return v.validateStringLen(sv)
	case util.StringSlice:
		for i := 0; i < value.Len(); i++ {
			sv := value.Index(i).String()
			if err := v.validateStringLen(sv); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("string len range validator can only used for string")
	}
}

func (v *stringLenRangeValidator) validateStringLen(s string) error {
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
	kind := util.Inspect(value.Type())
	switch kind {
	case util.Int:
		return v.validateValueRange(value.Int())
	case util.IntSlice:
		for i := 0; i < value.Len(); i++ {
			sv := value.Index(i)
			if err := v.validateValueRange(sv.Int()); err != nil {
				return err
			}
		}
		return nil
	case util.Uint:
		return v.validateValueRange(int64(value.Uint()))
	case util.UintSlice:
		for i := 0; i < value.Len(); i++ {
			sv := value.Index(i)
			if err := v.validateValueRange(int64(sv.Uint())); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("int range validator can only used for integers")
	}
}

func (v *intRangeValidator) validateValueRange(i int64) error {
	if i < v.min || i >= v.max {
		return fmt.Errorf("int value %v exceed the range limit[%v:%v)", i, v.min, v.max)
	}
	return nil
}

const (
	domainNameTag    = "isDomain="
	minTag           = "min="
	maxTag           = "max="
	minLenTag        = "minLen="
	maxLenTag        = "maxLen="
	optionsTag       = "options="
	optionsDelimiter = "|"
)

func buildValidator(fieldType reflect.Type, restTags []string) (Validator, error) {
	var minStr, maxStr string
	var minLenStr, maxLenStr string
	var options []string
	var isValueRangeCheck, isDomainCheck, isStringLenCheck, isOptionCheck bool
	for _, tag := range restTags {
		if strings.HasPrefix(tag, domainNameTag) {
			isDomainCheck = true
		} else if strings.HasPrefix(tag, minTag) {
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

	if minLenStr == "" && maxLenStr != "" {
		return nil, fmt.Errorf("has maxLen but not minLen")
	} else if minLenStr != "" && maxLenStr == "" {
		return nil, fmt.Errorf("has minLen but not maxLen")
	} else if minLenStr != "" && maxLenStr != "" {
		isStringLenCheck = true
	}
	isOptionCheck = len(options) > 0

	if minStr != "" && maxStr == "" {
		return nil, fmt.Errorf("has min but not max")
	} else if minStr == "" && maxStr != "" {
		return nil, fmt.Errorf("has max but not min")
	} else if minStr != "" && maxStr != "" {
		isValueRangeCheck = true
	}

	kind := util.Inspect(fieldType)
	switch kind {
	case util.Int, util.Uint:
		if isDomainCheck || isOptionCheck || isStringLenCheck {
			return nil, fmt.Errorf("domain, length range and option check only apply for string field")
		}

		if isValueRangeCheck {
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

	case util.String, util.StringSlice:
		if isValueRangeCheck {
			return nil, fmt.Errorf("range check only apply for integer")
		}
		checks := 0
		if isDomainCheck {
			checks += 1
		}
		if isOptionCheck {
			checks += 1
		}
		if isStringLenCheck {
			checks += 1
		}

		if checks > 1 {
			return nil, fmt.Errorf("domain, length range and option validation are conflict with each other")
		}

		if isDomainCheck {
			return newDomainNameValidator(), nil
		} else if isOptionCheck {
			return newOptionValidator(options), nil
		} else if isStringLenCheck {
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

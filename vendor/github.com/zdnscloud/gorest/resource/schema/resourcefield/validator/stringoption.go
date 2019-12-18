package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gorest/util"
)

const (
	optionsTag       = "options="
	optionsDelimiter = "|"
)

type optionValidator struct {
	options []string
}

type optionValidatorBuilder struct{}

func newOptionValidator(options []string) Validator {
	return &optionValidator{options: options}
}

func (v *optionValidator) Validate(val interface{}) error {
	value := reflect.ValueOf(val)
	kind := util.Inspect(value.Type())
	if kind != util.String {
		return fmt.Errorf("option apply to non-string type: %v", kind)
	}
	sv := value.String()
	if slice.SliceIndex(v.options, sv) == -1 {
		return fmt.Errorf("%s isn't included in options %v", sv, v.options)
	}
	return nil
}

func (b *optionValidatorBuilder) FromTags(tags []string) (Validator, error) {
	for _, tag := range tags {
		if strings.HasPrefix(tag, optionsTag) {
			options := strings.Split(strings.TrimPrefix(tag, optionsTag), optionsDelimiter)
			return newOptionValidator(options), nil
		}
	}
	return nil, nil
}

func (b *optionValidatorBuilder) SupportKind(kind util.Kind) bool {
	return kind == util.String ||
		kind == util.StringSlice ||
		kind == util.StringStringMap
}

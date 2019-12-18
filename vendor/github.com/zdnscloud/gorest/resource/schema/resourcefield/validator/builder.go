package validator

import (
	"reflect"

	"github.com/zdnscloud/gorest/util"
)

var builders []ValidatorBuilder = []ValidatorBuilder{
	&domainNameValidatorBuilder{},
	&stringLenRangeValidatorBuilder{},
	&intRangeValidatorBuilder{},
	&optionValidatorBuilder{},
}

func Build(fieldType reflect.Type, tags []string) ([]Validator, error) {
	var vs []Validator
	kind := util.Inspect(fieldType)
	for _, builder := range builders {
		if builder.SupportKind(kind) {
			v, err := builder.FromTags(tags)
			if err != nil {
				return nil, err
			}
			if v != nil {
				vs = append(vs, v)
			}
		}
	}
	return vs, nil
}

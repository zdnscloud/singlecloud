package validator

import (
	"github.com/zdnscloud/gorest/util"
)

type Validator interface {
	//validate each field is valid
	Validate(interface{}) error
}

type ValidatorBuilder interface {
	FromTags([]string) (Validator, error)
	SupportKind(util.Kind) bool
}

package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/zdnscloud/gorest/util"
)

const domainPrefix = "isDomain="

type domainNameValidator struct{}
type domainNameValidatorBuilder struct{}

var gDomainNameValidator Validator = &domainNameValidator{}
var _ ValidatorBuilder = &domainNameValidatorBuilder{}

const (
	dns1123LabelFmt           string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123SubdomainFmt       string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"
	DNS1123SubdomainMaxLength int    = 253
)

var dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")

func (v *domainNameValidator) Validate(val interface{}) error {
	value := reflect.ValueOf(val)
	kind := util.Inspect(value.Type())
	if kind != util.String {
		return fmt.Errorf("isDomain apply to non-string type: %v", kind)
	}
	return validateDomain(value.String())
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

func (b *domainNameValidatorBuilder) FromTags(tags []string) (Validator, error) {
	for _, tag := range tags {
		if strings.HasPrefix(tag, domainPrefix) {
			return gDomainNameValidator, nil
		}
	}
	return nil, nil
}

func (b *domainNameValidatorBuilder) SupportKind(kind util.Kind) bool {
	return kind == util.String ||
		kind == util.StringSlice ||
		kind == util.StringStringMap
}

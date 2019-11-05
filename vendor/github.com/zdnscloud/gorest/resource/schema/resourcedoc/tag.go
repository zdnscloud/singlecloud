package resourcedoc

import (
	"reflect"
	"strings"
)

const (
	requiredTag    = "required"
	optionsTag     = "options="
	descriptionTag = "description="
)

func DescriptionTag(tag reflect.StructTag) []string {
	var describe []string
	restTags := strings.Split(tag.Get("rest"), ",")
	for _, t := range restTags {
		if strings.HasPrefix(t, requiredTag) {
			describe = append(describe, requiredTag)
		}
		if strings.HasPrefix(t, descriptionTag) {
			descriptionVal := strings.TrimPrefix(t, descriptionTag)
			describe = append(describe, descriptionVal)
		}
	}
	return describe
}

func OptionsTag(tag reflect.StructTag) []string {
	restTags := strings.Split(tag.Get("rest"), ",")
	for _, t := range restTags {
		if strings.HasPrefix(t, optionsTag) {
			return strings.Split(strings.TrimPrefix(t, optionsTag), "|")
		}
	}
	return []string{}
}

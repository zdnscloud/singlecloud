package resourcefield

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	requiredTag = "required="
)

func fieldParseOptional(f Field, kind reflect.Kind, restTags []string) error {
	for _, tag := range restTags {
		if strings.HasPrefix(tag, requiredTag) {
			requiredVal := strings.TrimPrefix(tag, requiredTag)
			if requiredVal == "no" || requiredVal == "false" {
				f.SetRequired(false)
			} else if requiredVal == "yes" || requiredVal == "true" {
				f.SetRequired(true)
			} else {
				return fmt.Errorf("invalid require value %s", requiredVal)
			}
		}
	}

	return nil
}

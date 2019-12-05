package util

import (
	"strings"
)

func GuessPluralName(name string) string {
	if name == "" {
		return name
	}

	if strings.EqualFold(name, "Endpoints") {
		return name
	}

	if strings.HasSuffix(name, "s") || strings.HasSuffix(name, "ch") || strings.HasSuffix(name, "x") || strings.HasSuffix(name, "zh") || strings.HasSuffix(name, "sh") {
		return name + "es"
	}

	if strings.HasSuffix(name, "y") && len(name) > 2 && !strings.ContainsAny(name[len(name)-2:len(name)-1], "[aeiou]") {
		return name[0:len(name)-1] + "ies"
	}

	return name + "s"
}

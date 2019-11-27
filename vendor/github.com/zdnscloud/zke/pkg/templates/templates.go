package templates

import (
	"bytes"
	"strings"
	"text/template"
)

var templateFuncMap = template.FuncMap{
	"nindent":    nindent,
	"replace":    func(old, new, src string) string { return strings.Replace(src, old, new, -1) },
	"trimPrefix": func(prefix, src string) string { return strings.TrimPrefix(src, prefix) },
	"repeat":     func(count int, str string) string { return strings.Repeat(str, count) },
}

func nindent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return "\n" + pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

func CompileTemplateFromMap(tmplt string, configMap interface{}) (string, error) {
	out := new(bytes.Buffer)
	t := template.Must(template.New("compiled_template").Funcs(templateFuncMap).Parse(tmplt))
	if err := t.Execute(out, configMap); err != nil {
		return "", err
	}
	return out.String(), nil
}

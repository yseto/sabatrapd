package template

import (
	"bytes"
	"html/template"
)

func Parse(format string) error {
	funcmap := template.FuncMap{
		"read": func(key string) string {
			return "dummy"
		},
		"addr": func() string {
			return "dummy"
		},
	}
	_, err := template.New("").Funcs(funcmap).Parse(format)
	return err
}

func Execute(format string, pad map[string]string, addr string) (string, error) {
	funcmap := template.FuncMap{
		"read": func(key string) string {
			return pad[key]
		},
		"addr": func() string {
			return addr
		},
	}
	var tmpl = template.New("").Funcs(funcmap)

	var wr bytes.Buffer
	if err := template.Must(tmpl.Parse(format)).Execute(&wr, pad); err != nil {
		return "", err
	}

	return wr.String(), nil
}

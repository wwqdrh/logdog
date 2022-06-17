package api

import (
	"html/template"
	"io"
)

// bindatatemplate 方法
func ExecuteTemplate(wr io.Writer, name string, body []byte, data interface{}) error {
	tmpl := template.New(name)

	newTmpl, err := tmpl.Parse(string(body))
	if err != nil {
		return err
	}
	return newTmpl.Execute(wr, data)
}

package main

import (
	"text/template"

	"github.com/falling-sky/builder/tfuncs"
)

// MakeFuncMap returns a function map to use with text/template
func MakeFuncMap() template.FuncMap {
	f := make(template.FuncMap)
	f["include"] = tfuncs.Include
	f["process"] = tfuncs.Include
	return f
}

func main() {
}

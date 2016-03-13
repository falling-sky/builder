package tcache

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"text/template"

	"github.com/falling-sky/builder/fileutil"
)

// Tcache contains a map of templates, key=filename (minus root directory name)
type Tcache map[string]*template.Template

type TemplateData struct {
	Page        string
	GetRevision string
	Names       map[string]string
	Translated  map[string]string
}

// TopFiles returns the list of templates found at the top level of the
// scanned directory.  The intention of this is that we can convert/expand
// all top level files; and ignore anything below.
// This will let us stop explicitly having to indicate every file name
// in the builder; and let it be more dynamic.
func (tc Tcache) TopFiles() []string {
	ret := []string{}
	for k := range tc {
		if strings.Contains(k, "/") == false {
			ret = append(ret, k)
		}
	}
	sort.Strings(ret)
	return ret
}

// Files returns all files found in the template root directory, including in subdirectories.
func (tc Tcache) Files() []string {
	ret := []string{}
	for k := range tc {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

// New returns a map of templates, key=filename (minus root directory name)
func New(path string) (Tcache, error) {
	tc := make(Tcache)
	f, err := fileutil.FilesInDir(path)
	if err != nil {
		return tc, err
	}

	FuncMap := make(template.FuncMap)
	FuncMap["PROCESS"] = func(name string) (string, error) {
		log.Printf("PROCESS: %v\n", name)
		fname := path + "/" + name
		b, err := ioutil.ReadFile(fname)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	//log.Printf("%#v", f)
	for i, fn := range f {
		if strings.HasSuffix(fn, "~") {
			continue
		}
		if strings.Contains(fn, "/") {
			continue
		}
		log.Printf("i %v fn %v\n", i, fn)

		fname := path + "/" + fn
		t, err := template.New("new").Delims(`[%`, `%]`).Funcs(FuncMap).ParseFiles(fname)
		if err != nil {
			return tc, fmt.Errorf("Reading file %s: %s", fname, err)
		}
		tc[fn] = t
	}

	return tc, nil
}

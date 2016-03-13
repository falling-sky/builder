package tcache

import (
	"fmt"
	"log"
	"text/template"

	"github.com/falling-sky/builder/fileutil"
)

// Tcache contains a map of templates, key=filename (minus root directory name)
type Tcache map[string]*template.Template

// New returns a map of templates, key=filename (minus root directory name)
func New(path string) (Tcache, error) {
	tc := make(Tcache)
	f, err := fileutil.FilesInDir(path)
	if err != nil {
		return tc, err
	}
	//log.Printf("%#v", f)
	for i, fn := range f {
		log.Printf("i %v fn %v\n", i, fn)
		t, err := template.ParseFiles(path + "/" + fn)
		if err != nil {
			return tc, fmt.Errorf("Reading dir %s: %s", path, err)
		}
		tc[fn] = t
	}
	return tc, nil
}

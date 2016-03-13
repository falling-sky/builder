package tfuncs

import (
	"io/ioutil"
)

// Include will fetch the named file, and return the contents + error
// Really, this is just a string version of ioutil.ReadFile; but
// will potentially in the future offer caching.  Memory is cheap!
func Include(fn string) (string, error) {
	b, err := ioutil.ReadFile(fn)
	return string(b), err
}

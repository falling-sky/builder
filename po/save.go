package po

import (
	"bytes"
	"io/ioutil"
	"log"
	"strconv"
)

// Load a .PO file into memory.
func (f *File) Save(fn string) error {
	log.Printf("Generating %s\n", fn)
	f.ByID[""] = &Record{
		MsgID: "",
		MsgStr: `Project-Id-Version: PACKAGE VERSION
PO-Revision-Date: YEAR-MO-DA HO:MI +ZONE
Last-Translator: Unspecified Translator <jfesler+unspecified-translator@test-ipv6.com>
Language-Team: LANGUAGE <v6code@test-ipv6.com>
MIME-Version: 1.0
Content-Type: text/plain; charset=UTF-8
Content-Transfer-Encoding: 8bit`,
	}

	// Start new output buffer
	b := &bytes.Buffer{}

	// Prepend "" into the order
	f.InOrder = append([]string{""}, f.InOrder...)

	for _, str := range f.InOrder {
		r := f.ByID[str]
		if r.Comment != "" {
			b.WriteString("#: " + strconv.Quote(r.Comment) + "\n")
		}
		b.WriteString("msgid " + strconv.Quote(r.MsgID) + "\n")
		b.WriteString("msgstr" + strconv.Quote(r.MsgStr) + "\n")
		b.WriteString("\n")
	}
	err := ioutil.WriteFile(fn, b.Bytes(), 0644)
	return err

}

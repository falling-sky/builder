package po

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/falling-sky/builder/fileutil"
)

// Record is a single text translated
type Record struct {
	Comment string
	MsgID   string
	MsgStr  string
}

// MapStringRecord maps original strings to Records
type MapStringRecord map[string]*Record
type MapHeaders map[string]string

// File contains the map of strings for this translation.
type File struct {
	ByID       MapStringRecord
	Headers    MapHeaders
	Language   string
	Translated int
	OutOf      int
}
type MapStringFile map[string]*File
type Files struct {
	Pot        *File
	ByLanguage MapStringFile
}

var example = `
#: faq_whyipv6.html
msgid "Note this is in addition to any NAT you do at home."
msgstr ""

#: faq_whyipv6.html
msgid "Q: So, why worry? NAT will work, right? I use NAT at home today after all.."
msgstr ""
`

func unquote(s string) (string, error) {
	if len(s) == 0 {
		return s, nil
	}
	s = strings.TrimSpace(s)
	return strconv.Unquote(s)
}

func parseChunk(chunk string) (*Record, error) {
	lines := strings.Split(chunk, "\n")
	token := ""
	hash := make(map[string]string)

	// Convert chunk to hash (map) table of token/string
	// taking into account continuation lines, and also
	// unquoting strings
	for _, line := range lines {
		if len(line) == 0 {
			continue // Last chunk may have this empty line.
		}
		remainder := line
		//log.Printf("line:'%v'\n", line)
		if line[0:1] != "\"" {
			parts := strings.SplitN(line, " ", 2)
			token = parts[0]
			remainder = parts[1]
		}
		if remainder[0:1] == "\"" {
			replacement, err := strconv.Unquote(remainder)
			if err != nil {
				return nil, fmt.Errorf("while unquoting string %v, error: %v", remainder, err)
			}
			remainder = replacement
		}
		if _, ok := hash[token]; ok {
			hash[token] = hash[token] + remainder
		} else {
			hash[token] = remainder
		}
	}

	// Convert hash to a record with strict
	// field names and types
	record := &Record{}
	record.Comment = hash["#:"]
	record.MsgID = hash["msgid"]
	record.MsgStr = hash["msgstr"]
	return record, nil
}

func parseHeaders(s string) (MapHeaders, error) {
	//	log.Printf("parseHeaders: %s", s)
	headerLines := strings.Split(s, "\n")
	h := make(MapHeaders)
	for _, line := range headerLines {
		//log.Printf("header line='%s'\n", line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			h[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	//log.Printf("Parsed Headers: %#v", h)
	return h, nil
}

// Load a .PO file into memory.
func Load(fn string) (*File, error) {

	f := &File{}
	f.ByID = make(MapStringRecord)

	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	chunks := strings.Split(string(b), "\n\n")
	if len(chunks) < 2 {
		return nil, fmt.Errorf("Bad format (or perhaps CR/LF) %v", fn)
	}

	for _, chunk := range chunks {
		//	log.Printf("Chunk: %s", chunk)
		record, err := parseChunk(chunk)
		if err != nil {
			return nil, fmt.Errorf("Parsing chunk from %s: %s", fn, err)
		}
		if record.MsgStr != "" || record.MsgID != "" {
			//log.Printf("Parsed Chunk: %#v\n", record)
			f.ByID[record.MsgID] = record
		}
	}

	// Parse Headers
	rootRecord := f.ByID[""]
	if rootRecord == nil {
		return nil, fmt.Errorf("File %v missing root record", fn)
	}
	f.Headers, err = parseHeaders(rootRecord.MsgStr)
	if err != nil {
		return nil, fmt.Errorf("File %s parsing header: %s", fn, err)
	}
	f.Language = f.Headers["Language"]
	if f.Language == "" {
		if strings.HasSuffix(fn, ".pot") == false {
			return nil, fmt.Errorf("File %v missing Language: header", fn)
		}
	}

	//log.Printf("%#v\n", f)

	return f, nil
}

func (combined *Files) Languages() []string {
	ret := []string{}

	for k := range combined.ByLanguage {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

// LoadAll loads a .pot file, and a directory of .po files.
// The .pot file is mostly used for statistics.
func LoadAll(potfn string, root string) (*Files, error) {
	combined := &Files{}
	combined.ByLanguage = make(MapStringFile)

	po, err := Load(potfn)
	if err != nil {
		return nil, err
	}
	combined.Pot = po

	ls, err := fileutil.FilesInDir(root)
	if err != nil {
		return nil, err
	}
	for _, f := range ls {
		fn := root + "/" + f
		if strings.HasSuffix(fn, ".po") {
			log.Printf("we should load: %v\n", fn)
			p, err := Load(fn)
			if err != nil {
				return nil, err
			}

			for k := range po.ByID {
				p.OutOf++
				if found, ok := p.ByID[k]; ok {
					if found.MsgStr != "" && found.MsgStr != k {
						//	log.Printf("MsgStr=%v\nk=%v\n\n", found.MsgStr, k)
						p.Translated++
					}
				}

			}

			combined.ByLanguage[p.Language] = p

		}

	}

	return combined, nil
}

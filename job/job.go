package job

import (
	"bytes"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/falling-sky/builder/fileutil"
	"github.com/falling-sky/builder/gitinfo"
	"github.com/falling-sky/builder/po"
)

// rePROCESS matches on   [% PROCESS "filename" %]
// and captures the entire template directivel as well as the inside filename.
var rePROCESS = regexp.MustCompile(`\[\%\s*PROCESS\s*"(.*?)"\s*\%\]`)
var reTRANSLATE = regexp.MustCompile(`(?ms){{(.*?)}}`)

type QueueItem struct {
	Filename     string
	PoFile       *po.File
	PostProcess  func(string, string) error
	InputDir     string
	OutputDir    string
	Data         *TemplateData
	EscapeQuotes bool
}

type QueueTracker struct {
	Channel chan *QueueItem
	WG      *sync.WaitGroup
}

type TemplateData struct {
	GitInfo  *gitinfo.GitInfo
	PoMap    po.MapStringFile
	Locale   string
	Lang     string
	LangUC   string
	Basename string
}

type ParsedCacheType struct {
	lock   sync.RWMutex
	byname map[string]string
}

var ParsedCache ParsedCacheType

func init() {
	ParsedCache.byname = make(map[string]string)
}

func GrabContent(qi *QueueItem) string {
	topName := qi.InputDir + "/" + qi.Filename

	grab := func(fn string) string {
		fullname := qi.InputDir + "/" + fn
		c, err := fileutil.ReadFileWithCache(fullname)
		if err != err {
			log.Fatalf("tried to load %s (via %s): %s", fullname, topName, err)
		}
		// log.Printf("read %v (%v bytes)\n", fullname, len(c))
		return c
	}

	content := grab(qi.Filename)
	// Do we see PROCESS lines?
	for {
		matches := rePROCESS.FindStringSubmatch(content)
		if len(matches) == 0 {
			break
		}
		if len(matches) < 2 {
			log.Fatalf("I don't know what happened, but %s is interesting", matches[0])
		}
		wrapperString := matches[0]
		insideName := matches[1]
		newContent := grab(insideName)
		content = strings.Replace(content, wrapperString, newContent, -1)
	}
	return content
}

func ProcessTemplate(qi *QueueItem, content string) string {
	topName := qi.InputDir + "/" + qi.Filename

	// Do we need any custom functions?
	FuncMap := make(template.FuncMap)
	FuncMap["EXAMPLE"] = func(name string) (string, error) {
		//log.Printf("PROCESS: %v\n", name)
		return "", nil
	}

	// Parse the template.  Just looks for markers and implied commands.
	root := template.New(qi.Filename).Delims(`[%`, `%]`).Funcs(FuncMap)
	tmpl, err := root.Parse(content)
	if err != nil {
		log.Fatalf("Parsing template for %v: %v", topName, err)
	}

	// Execute the template.
	wr := &bytes.Buffer{}
	err = tmpl.Execute(wr, qi.Data)
	if err != nil {
		log.Fatalf("Executing template for %v: %v", topName, err)
	}

	return string(wr.Bytes())
}

func TranslateContent(qi *QueueItem, content string) string {
	for {

		matches := reTRANSLATE.FindStringSubmatch(content)
		if len(matches) == 0 {
			break
		}
		if len(matches) < 2 {
			log.Fatalf("I don't know what happened, but %s is interesting", matches[0])
		}
		wrapperString := matches[0]
		insideName := matches[1]

		//	log.Printf("grabbing %v\n", insideName)
		newContent := qi.PoFile.Translate(insideName, qi.EscapeQuotes)

		log.Printf("Replacing %s with %s\n", wrapperString, newContent)

		content = strings.Replace(content, wrapperString, newContent, -1)

	}
	return content
}

func RunJob(qi *QueueItem) {
	t0 := time.Now()
	defer func() {
		t1 := time.Now()
		dur := t1.Sub(t0)
		ms := int64(dur / time.Millisecond)
		if ms > 2 {
			log.Printf("Spent %v ms\n", ms)
		}
	}()
	log.Printf("RunJob Filename=%s PoLang=%s\n", qi.Filename, qi.PoFile.Language)
	readFilename := qi.InputDir + "/" + qi.Filename
	writeFilename := qi.OutputDir + "/" + qi.Filename
	_ = writeFilename

	var content string
	ParsedCache.lock.Lock()
	if c, ok := ParsedCache.byname[readFilename]; ok {
		// log.Printf("cached: %s", readFilename)
		content = c
	} else {
		// log.Printf("not cached: %s", readFilename)
		content = GrabContent(qi)
		content = ProcessTemplate(qi, content)
		ParsedCache.byname[readFilename] = content
	}
	ParsedCache.lock.Unlock()

	// TODO process translations

	content = TranslateContent(qi, content)

	outname := qi.OutputDir + "/" + qi.Filename + "." + qi.PoFile.Language
	err := ioutil.WriteFile(outname, []byte(content), 0755)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s etc (%v bytes)\n", outname, len(content))

	//	ioutil.WriteFile(writeFilename, []byte(content), 0755)
	//
	// This is where the dreams are made.  Or the nightmares.

}

func (qt *QueueTracker) RunQueue() {
	for {
		job, ok := <-qt.Channel
		if ok {
			RunJob(job)
			qt.WG.Done()
		} else {
			return
		}
	}
}

func (qt *QueueTracker) Add(qi *QueueItem) {
	qt.WG.Add(1)
	qt.Channel <- qi
}

func (qt *QueueTracker) Wait() {
	log.Printf("WAITING\n")
	qt.WG.Wait()
}

func (qt *QueueTracker) Close() {
	close(qt.Channel)
	qt.Wait()
}

func StartQueue() *QueueTracker {
	qt := &QueueTracker{}
	qt.Channel = make(chan *QueueItem, 1000)
	qt.WG = &sync.WaitGroup{}
	go qt.RunQueue()

	return qt
}

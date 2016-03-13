package job

import (
	"bytes"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/falling-sky/builder/fileutil"
	"github.com/falling-sky/builder/gitinfo"
	"github.com/falling-sky/builder/po"
)

// rePROCESS matches on   [% PROCESS "filename" %]
// and captures the entire template directivel as well as the inside filename.
var rePROCESS = regexp.MustCompile(`\[\%\s*PROCESS\s*"(.*?)"\s*\%\]`)

type QueueItem struct {
	Filename    string
	PoFile      *po.File
	PostProcess func(string, string) error
	InputDir    string
	OutputDir   string
	Data        *TemplateData
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

func GrabContent(qi *QueueItem) string {
	topName := qi.InputDir + "/" + qi.Filename

	grab := func(fn string) string {
		fullname := qi.InputDir + "/" + fn
		c, err := fileutil.ReadFileWithCache(fullname)
		if err != err {
			log.Fatalf("tried to load %s (via %s): %s", fullname, topName, err)
		}
		log.Printf("read %v (%v bytes)\n", fullname, len(c))
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
		log.Printf("grabbing %v\n", insideName)
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

func RunJob(qi *QueueItem) {
	log.Printf("RunJob Filename=%s PoLang=%s\n", qi.Filename, qi.PoFile.Language)

	writeFilename := qi.OutputDir + "/" + qi.Filename
	_ = writeFilename

	content := GrabContent(qi)
	content = ProcessTemplate(qi, content)
	// TODO process translations

	ioutil.WriteFile(writeFilename, []byte(content), 0755)
	log.Printf("wrote %s etc (%v bytes)\n", writeFilename, len(content))
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
	qt.Channel = make(chan *QueueItem, 10)
	qt.WG = &sync.WaitGroup{}
	go qt.RunQueue()
	return qt
}

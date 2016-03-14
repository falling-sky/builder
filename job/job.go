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

	"github.com/falling-sky/builder/config"
	"github.com/falling-sky/builder/fileutil"
	"github.com/falling-sky/builder/gitinfo"
	"github.com/falling-sky/builder/po"
)

// rePROCESS matches on   [% PROCESS "filename" %]
// and captures the entire template directivel as well as the inside filename.
var rePROCESS = regexp.MustCompile(`\[\%\s*PROCESS\s*"(.*?)"\s*\%\]`)
var reTRANSLATE = regexp.MustCompile(`(?ms){{(.*?)}}`)

// QueueItem represents a single job to be queued, and ran as capacity allows.
// This is so we can generate the work list up front; and then pace out the work
// based on number of avaialble CPUs.
type QueueItem struct {
	Config       *config.Record
	RootDir      string
	Filename     string
	PoFile       *po.File
	PostProcess  func(string, string) error
	Data         *TemplateData
	EscapeQuotes bool
	MultiLocale  bool
}

// QueueTracker is an object for managing QueueItem jobs.
type QueueTracker struct {
	Channel chan *QueueItem
	WG      *sync.WaitGroup
}

// TemplateData is passed when adding the job to the queue.
// This is used by Go's text/template to extract info before expansion.
type TemplateData struct {
	GitInfo  *gitinfo.GitInfo
	PoMap    po.MapStringFile
	Locale   string
	Lang     string
	LangUC   string
	Basename string
}

// ParsedCacheType provides properly mutex locked cache access to
// the expanded (but untranslated) templates.
type ParsedCacheType struct {
	lock   sync.RWMutex
	byname map[string]string
}

// ParsedCache holds the actual cache of expanded (but not translated) templates.
var ParsedCache ParsedCacheType

func init() {
	ParsedCache.byname = make(map[string]string)
}

// GrabContent grabs a file.  Takes into account the QueueItem variables
// such as the iput directory path.  The file is cached for future requests.
func GrabContent(qi *QueueItem) string {
	topName := qi.RootDir + "/" + qi.Filename

	grab := func(fn string) string {
		fullname := qi.RootDir + "/" + fn
		c, err := fileutil.ReadFile(fullname)
		if err != nil {
			log.Fatalf("tried to load %s (via %s): %s", fullname, topName, err)
		}
		//		log.Printf("read %v (%v bytes)\n", fullname, len(c))
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

// ProcessTemplate runs text.Template against the given text.
// Note we use [% %]  for text.Template directorives, since these
// are fewer than translations. And we prefer to do translations
// without the template ugliness.
func ProcessTemplate(qi *QueueItem, content string) string {
	topName := qi.RootDir + "/" + qi.Filename

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

// TranslateContent  looks for {{ text }} and replaces it with
// either translated text, or the original text.
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

		//	log.Printf("Replacing %s with %s\n", wrapperString, newContent)

		content = strings.Replace(content, wrapperString, newContent, -1)

	}
	return content
}

// RunJob takes a single QueueItem, and expands, translates, optimizes,
// and writes files for that single file for a single language.  These are spoon-fed
// by RunQueue.
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
	readFilename := qi.RootDir + "/" + qi.Filename
	writeFilename := qi.Config.Directories.OutputDir + "/" + qi.Filename
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

	outname := qi.Config.Directories.OutputDir + "/" + qi.Filename + "." + qi.PoFile.Language
	if qi.MultiLocale == false {
		outname = qi.Config.Directories.OutputDir + "/" + qi.Filename
	}
	err := ioutil.WriteFile(outname, []byte(content), 0755)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s etc (%v bytes)\n", outname, len(content))

	//	ioutil.WriteFile(writeFilename, []byte(content), 0755)
	//
	// This is where the dreams are made.  Or the nightmares.

}

// RunQueue is a goroutine that listens to a channel for jobs.
// If jobs are accepted, they are given to RunJob.
func (qt *QueueTracker) RunQueue() {
	for {
		job, ok := <-qt.Channel
		if ok {
			RunJob(job)  // Run the job.
			qt.WG.Done() // Decrement WaitGroup counter
		} else {
			return
		}
	}
}

// Add a job to the queue.  Sends it to the channel.
func (qt *QueueTracker) Add(qi *QueueItem) {
	qt.WG.Add(1)     // Increment the WaitGroup counter.
	qt.Channel <- qi // Put the job in the queue.
}

// Wait will wait for all existing jobs to finish.
func (qt *QueueTracker) Wait() {
	log.Printf("WAITING\n")
	qt.WG.Wait()
}

// StartQueue will start a goroutine for jobs, and return
// a handle to be used for adding and waiting on jobs.
func StartQueue() *QueueTracker {
	qt := &QueueTracker{}
	qt.Channel = make(chan *QueueItem, 1000)
	qt.WG = &sync.WaitGroup{}
	go qt.RunQueue()
	go qt.RunQueue()
	go qt.RunQueue()
	go qt.RunQueue()

	return qt
}

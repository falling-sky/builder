package job

import (
	"bytes"
	"io/ioutil"
	"log"
	"sync"

	"github.com/falling-sky/builder/po"
	"github.com/falling-sky/builder/tcache"
)

type QueueItem struct {
	Filename     string
	PoFile       *po.File
	Templates    tcache.Tcache
	TemplateData *tcache.TemplateData
	PostProcess  func(string, string) error
	OutputDir    string
}

type QueueTracker struct {
	Channel chan *QueueItem
	WG      *sync.WaitGroup
}

func RunJob(qi *QueueItem) {
	log.Printf("RunJob Filename=%s PoLang=%s\n", qi.Filename, qi.PoFile.Language)

	// Do stuff here.

	// This is where the dreams are made.  Or the nightmares.

	// Create a memory buffer to store this intention
	templateOutputBuffer := &bytes.Buffer{}
	tmpl := qi.Templates[qi.Filename]
	if tmpl == nil {
		log.Fatalf("should not have been able to ask for template %v when it is not loaded", qi.Filename)
	}
	//log.Printf("%#v\n", tmpl)

	err := tmpl.Execute(templateOutputBuffer, qi.TemplateData)
	if err != nil {
		log.Fatalf("Processing template named %#v, Execute says: %v", qi.Filename, err)
	}

	filename := qi.OutputDir + "/" + qi.Filename
	log.Printf("write %s (%v bytes)\n", filename, templateOutputBuffer.Len())
	err = ioutil.WriteFile(filename, templateOutputBuffer.Bytes(), 0755)
	if err != nil {
		log.Fatal(err)
	}

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

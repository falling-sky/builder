package job

import (
	"log"
	"sync"

	"github.com/falling-sky/builder/po"
	"github.com/falling-sky/builder/tcache"
)

type QueueItem struct {
	Filename    string
	PoFile      *po.File
	Tcache      tcache.Tcache
	PostProcess func(string, string) error
}

type QueueTracker struct {
	Channel chan *QueueItem
	WG      *sync.WaitGroup
}

func RunJob(qi *QueueItem) {
	log.Printf("RunJob Filename=%s PoLang=%s\n", qi.Filename, qi.PoFile.Language)
	// Do stuff here.
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

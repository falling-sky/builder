package main

import (
	"flag"
	"log"
	"text/template"

	"github.com/falling-sky/builder/job"
	"github.com/falling-sky/builder/po"
	"github.com/falling-sky/builder/postprocess"
	"github.com/falling-sky/builder/tcache"
	"github.com/falling-sky/builder/tfuncs"
)

var templateDir = flag.String("templatedir", "templates", "Location of dir containing html, css, js subdirs")
var poDir = flag.String("poDir", "translations", "Location of falling-sky.pot, and downloaded translations")
var outputDir = flag.String("outputdir", "output", "Location to place output into")

type postType struct {
	extension   string
	postprocess func(string, string) error
}

var postTable = []postType{
	{"html", postprocess.PostProcessHTML},
	{"css", postprocess.PostProcessCSS},
	{"js", postprocess.PostProcessJS},
}

// MakeFuncMap returns a function map to use with text/template
func MakeFuncMap() template.FuncMap {
	f := make(template.FuncMap)
	f["include"] = tfuncs.Include
	f["process"] = tfuncs.Include
	return f
}

func must(a interface{}, b error) interface{} {
	if b != nil {
		log.Fatal(b)
	}
	return a
}

// I have languages loaded
// I have templates loaded
// I need some metadata loaded

// I need to run once per language per file

// I need to build a queue of jobs.
// I need to consume the queue of jobs.
// I need to indicate when I'm done with the jobs.

// I can use a channel for the write queue.
// I can use a waitgroup (Shared) to detect when done.

// I want to limit the parallel jobs to number of CPUs.

func main() {
	flag.Parse()

	jobTracker := job.StartQueue()

	languages, err := po.LoadAll(*poDir+"/falling-sky.pot", *poDir+"/dl")
	if err != nil {
		log.Fatal(err)
	}

	for _, tt := range postTable {
		template, err := tcache.New(*templateDir + "/" + tt.extension)
		if err != nil {
			log.Fatal(err)
		}
		files := template.TopFiles()
		for _, file := range files {
			for _, pofile := range languages.ByLanguage {
				job := &job.QueueItem{
					Filename: file,
					PoFile:   pofile,
				}
				jobTracker.Add(job)
			}

		}
	}
	jobTracker.Wait()

}

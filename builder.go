package main

import (
	"flag"
	"log"
	"strings"

	"github.com/falling-sky/builder/fileutil"
	"github.com/falling-sky/builder/gitinfo"
	"github.com/falling-sky/builder/job"
	"github.com/falling-sky/builder/po"
	"github.com/falling-sky/builder/postprocess"
)

var templateDir = flag.String("templatedir", "templates", "Location of dir containing html, css, js subdirs")
var poDir = flag.String("poDir", "translations", "Location of falling-sky.pot, and downloaded translations")
var outputDir = flag.String("outputdir", "output", "Location to place output into")

type postType struct {
	extension   string
	postprocess func(string, string) error
	escapequote bool
}

var postTable = []postType{
	{"html", postprocess.HTML, false},
	{"css", postprocess.CSS, false},
	{"js", postprocess.JS, true},
}

func main() {
	flag.Parse()

	// Start the job queue for templates
	jobTracker := job.StartQueue()

	// Load all langauges, calculate all percentages of completion.
	languages, err := po.LoadAll(*poDir+"/falling-sky.pot", *poDir+"/dl")
	if err != nil {
		log.Fatal(err)
	}

	// Grab this just once.
	cachedGitInfo := gitinfo.GetGitInfo()

	for _, tt := range postTable {
		inputDir := *templateDir + "/" + tt.extension
		files, err := fileutil.FilesInDirNotRecursive(inputDir)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("files: %#v\n", files)

		for _, file := range files {
			if strings.HasSuffix(file, tt.extension) {
				for locale, pofile := range languages.ByLanguage {

					// Build up what we need to know about the project, that
					// the templates will ask about.
					td := &job.TemplateData{}
					td.GitInfo = cachedGitInfo
					td.PoMap = languages.ByLanguage
					td.Locale = locale
					p := strings.Split(td.Locale, "_")
					td.Lang = p[0]
					td.LangUC = strings.ToUpper(td.Lang)

					p = strings.Split(file, ".")
					td.Basename = p[0]

					/*
						if locale != "de_DE" {
							continue
						}
					*/

					job := &job.QueueItem{
						Filename:     file,
						PoFile:       pofile,
						InputDir:     inputDir,
						OutputDir:    *outputDir,
						EscapeQuotes: tt.escapequote,
						Data:         td,
					}
					jobTracker.Add(job)
				}
			}
		}
	}
	// Wait for all jobs to finish
	jobTracker.Wait()

}

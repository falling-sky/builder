package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/falling-sky/builder/config"
	"github.com/falling-sky/builder/fileutil"
	"github.com/falling-sky/builder/gitinfo"
	"github.com/falling-sky/builder/job"
	"github.com/falling-sky/builder/po"
	"github.com/falling-sky/builder/postprocess"
)

var configFileName = flag.String("config", "", "config file location (see --example)")
var configHelp = flag.Bool("example", false, "Dump a configuration example to the screen.")

type postType struct {
	extension   string
	postprocess func(string, string) error
	escapequote bool
	multilocale bool
}

var postTable = []postType{
	{"html", postprocess.HTML, false, true},
	{"css", postprocess.CSS, false, true},
	{"js", postprocess.JS, true, true},
	{"php", postprocess.CSS, false, false},
}

func main() {
	flag.Parse()

	if *configHelp {
		fmt.Println(config.Example())
		os.Exit(0)
	}
	conf, err := config.Load(*configFileName)
	if err != nil {
		log.Fatal(err)
	}

	// Start the job queue for templates
	jobTracker := job.StartQueue()

	// Load all langauges, calculate all percentages of completion.
	languages, err := po.LoadAll(conf.Directories.PoDir+"/falling-sky.pot", conf.Directories.PoDir+"/dl")
	if err != nil {
		log.Fatal(err)
	}
	languages.Pot.Language = "en_US"

	// Grab this just once.
	cachedGitInfo := gitinfo.GetGitInfo()

	for _, tt := range postTable {
		inputDir := conf.Directories.PoDir + "/" + tt.extension
		files, err := fileutil.FilesInDirNotRecursive(inputDir)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("files: %#v\n", files)

		// Wrapper for launch jobs, gets all the variables into place and in scope
		launcher := func(file string, locale string, pofile *po.File) {

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
				Config:       conf,
				Filename:     file,
				PoFile:       pofile,
				EscapeQuotes: tt.escapequote,
				Data:         td,
				MultiLocale:  tt.multilocale,
			}
			jobTracker.Add(job)

		}

		// Start launching specific jobs
		for _, file := range files {
			if strings.HasSuffix(file, tt.extension) {
				launcher(file, "en_US", languages.Pot)
				if tt.multilocale {
					for locale, pofile := range languages.ByLanguage {
						launcher(file, locale, pofile)
					}
				}
			}
		}
	}
	// Wait for all jobs to finish
	jobTracker.Wait()

}

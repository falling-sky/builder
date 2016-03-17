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
	"github.com/falling-sky/builder/signature"
)

var configFileName = flag.String("config", "", "config file location (see --example)")
var configHelp = flag.Bool("example", false, "Dump a configuration example to the screen.")

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if *configHelp {
		fmt.Println(config.Example())
		os.Exit(0)
	}
	conf, err := config.Load(*configFileName)
	if err != nil {
		log.Fatal(err)
	}

	var postTable = []job.PostInfoType{
		{"css", ".css", conf.Processors.CSS, false, true, true},
		{"js", ".css", conf.Processors.JS, true, true, true},
		{"html", ".html", conf.Processors.HTML, false, true, true},
		{"php", ".php", conf.Processors.PHP, false, false, false},
		{"apache", ".htaccess", conf.Processors.Apache, false, false, false},
		{"apache", ".example", conf.Processors.Apache, false, false, false},
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
		inputDir := conf.Directories.TemplateDir + "/" + tt.Directory
		files, err := fileutil.FilesInDirNotRecursive(inputDir)
		if err != nil {
			log.Fatal(err)
		}
		//	log.Printf("files: %#v\n", files)

		rootDir := conf.Directories.TemplateDir + "/" + tt.Directory
		addLanguages := languages.ApacheAddLanguage()
		signature := signature.ScanDir(rootDir, addLanguages)

		// Wrapper for launch jobs, gets all the variables into place and in scope
		launcher := func(file string, locale string, pofile *po.File) {

			// Build up what we need to know about the project, that
			// the templates will ask about.
			td := &job.TemplateData{
				GitInfo:      cachedGitInfo,
				PoMap:        languages.ByLanguage,
				Locale:       pofile.GetLocale(),
				Lang:         pofile.GetLang(),
				LangUC:       pofile.GetLangUC(),
				Basename:     strings.Split(file, ".")[0],
				AddLanguage:  addLanguages,
				DirSignature: signature,
			}

			job := &job.QueueItem{
				Config:   conf,
				RootDir:  rootDir,
				Filename: file,
				PoFile:   pofile,
				Data:     td,
				PostInfo: tt,
			}
			jobTracker.Add(job)

		}

		// Start launching specific jobs
		for _, file := range files {
			if strings.HasSuffix(file, tt.Extension) {
				//		log.Printf("file=%s\n", file)
				launcher(file, "en_US", languages.NewPot)
				if tt.MultiLocale {
					for locale, pofile := range languages.ByLanguage {
						launcher(file, locale, pofile)
					}
				}
			}
		}
	}
	// Wait for all jobs to finish
	jobTracker.Wait()

	err = languages.NewPot.Save(conf.Directories.PoDir + "/falling-sky.newpot")
	if err != nil {
		log.Fatal(err)
	}

}

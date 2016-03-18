package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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

func copyImages(source string, dest string) {
	log.Printf("copyImages(%s,%s)\n", source, dest)
	files, err := fileutil.FilesInDirNotRecursive(source)
	if err != nil {
		log.Fatal(err)
	}
	os.MkdirAll(dest, 0755)
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if !(ext == ".png" || ext == ".gif" || ext == ".jpg" || ext == ".jpeg") {
			continue
		}
		log.Printf("copyImages(%s,%s) (%s)\n", source, dest, f)
		b, e := ioutil.ReadFile(source + "/" + f)
		if e != nil {
			log.Fatal(err)
		}
		e = ioutil.WriteFile(dest+"/"+f, b, 0644)
		if e != nil {
			log.Fatal(err)
		}
	}
}

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
		{
			Directory:   "css",
			Extension:   ".css",
			PostProcess: conf.Processors.CSS,
			EscapeQuote: false,
			MultiLocale: true,
			Compress:    true,
		},
		{
			Directory:   "js",
			Extension:   ".css",
			PostProcess: conf.Processors.JS,
			EscapeQuote: true,
			MultiLocale: true,
			Compress:    true,
		},
		{
			Directory:   "html",
			Extension:   ".html",
			PostProcess: conf.Processors.HTML,
			EscapeQuote: false,
			MultiLocale: true,
			Compress:    true,
		},
		{
			Directory:   "php",
			Extension:   ".php",
			PostProcess: conf.Processors.PHP,
			EscapeQuote: false,
			MultiLocale: false,
			Compress:    false,
		},
		{
			Directory:   "apache",
			Extension:   ".htaccess",
			PostProcess: conf.Processors.Apache,
			EscapeQuote: false,
			MultiLocale: false,
			Compress:    false,
		},
		{
			Directory:   "apache",
			Extension:   ".example",
			PostProcess: conf.Processors.Apache,
			EscapeQuote: false,
			MultiLocale: false,
			Compress:    false,
		},
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

	// Wait for all process jobs to finish
	jobTracker.Wait()

	// Copy images
	copyImages(conf.Directories.TemplateDir+"/images", conf.Directories.OutputDir+"/images")
	copyImages(conf.Directories.TemplateDir+"/images", conf.Directories.OutputDir+"/images-nc")

	err = languages.NewPot.Save(conf.Directories.PoDir + "/falling-sky.newpot")
	if err != nil {
		log.Fatal(err)
	}

}

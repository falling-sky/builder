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

func copyHelper(source string, dest string, fn func(string) ([]string, error)) {
	files, err := fn(source)
	if err != nil {
		log.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, f := range files {
		if strings.HasSuffix(f, "~") {
			continue // Skip editor backups
		}
		log.Printf("copy %s/%s to %s/%s\n", source, f, dest, f)

		// Read the file.
		b, e := ioutil.ReadFile(source + "/" + f)
		if e != nil {
			log.Fatal(err)
		}

		// Create directory, if needed.
		dir := filepath.Dir(dest + "/" + f)
		if _, ok := seen[dir]; ok == false {
			seen[dir] = true
			os.MkdirAll(dir, 0755)
		}

		// Write the file.
		e = ioutil.WriteFile(dest+"/"+f, b, 0644)
		if e != nil {
			log.Fatal(err)
		}
	}
}

func copyFiles(source string, dest string) {
	log.Printf("copyFiles(%s,%s)\n", source, dest)
	copyHelper(source, dest, fileutil.FilesInDirNotRecursive)
}

func copyFilesAll(source string, dest string) {
	log.Printf("copyFiles(%s,%s)\n", source, dest)
	copyHelper(source, dest, fileutil.FilesInDirRecursive)
}

func prepOutput(dir string) {
	log.Printf("Prepping %s\n", dir)
	if dir == "" {
		log.Fatal("dir empty, unexpected")
	}
	os.MkdirAll(dir, 0755)   // Make sure it exists, so that RemoveAll won't fail
	err := os.RemoveAll(dir) // Remove all - including old files, subdirs, etc.
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(dir, 0755) // Make sure the directory now exists, for real.
	if err != nil {
		log.Fatal(err)
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

	prepOutput(conf.Directories.OutputDir)

	var postTable = []job.PostInfoType{
		{
			Directory:   "css",
			Extension:   ".css",
			PostProcess: conf.Processors.CSS,
			EscapeQuote: false,
			MultiLocale: false,
			Compress:    true,
		},
		{
			Directory:   "js",
			Extension:   ".js",
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
	jobTracker := job.StartQueue(conf.Options.MaxThreads)

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
	copyFiles(conf.Directories.ImagesDir, conf.Directories.OutputDir+"/images")
	copyFiles(conf.Directories.ImagesDir, conf.Directories.OutputDir+"/images-nc")
	copyFilesAll(conf.Directories.TransparentDir, conf.Directories.OutputDir+"/transparent")

	// A couple last minute symlinks
	os.Symlink(".", conf.Directories.OutputDir+"/isp")
	os.Symlink(".", conf.Directories.OutputDir+"/helpdesk")

	err = languages.NewPot.Save(conf.Directories.PoDir + "/falling-sky.newpot")
	if err != nil {
		log.Fatal(err)
	}

}

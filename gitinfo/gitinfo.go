package gitinfo

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

type GitInfo struct {
	RevisionCount  string
	ProjectVersion string
	Version        string
	Date           string
	Repository     string
}

func GetGitInfo() *GitInfo {
	gi := &GitInfo{}
	gi.RevisionCount = GitRevisionCount()
	gi.ProjectVersion = GitProjectVersion()
	gi.Version = GitVersion()
	gi.Date = GitDate()
	gi.Repository = GitRepository()
	return gi
}

func GitRevisionCount() string {
	cmd := exec.Command("git", "log", "--oneline")
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("running %#v %#v: %v", cmd.Path, cmd.Args, err)
	}
	s := strings.TrimSpace(string(b))
	lines := strings.Split(s, "\n")
	return fmt.Sprintf("%v", len(lines))
}

func GitProjectVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--long")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "x.notags"
		//log.Fatalf("running %#v %#v: %v", cmd.Path, cmd.Args, err)
	}
	s := strings.TrimSpace(string(b))
	return s
}

func GitVersion() string {
	s := GitProjectVersion()
	parts := strings.Split(s, "-")
	version := fmt.Sprintf("%v.%v", parts[0], GitRevisionCount())
	return version
}

func GitDate() string {
	cmd := exec.Command("env", "TZ=UTC", "git", "log", "-1", `--format=%cd`)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("running %#v %#v: %v", cmd.Path, cmd.Args, err)
	}
	s := strings.TrimSpace(string(b))
	return s
}

func GitRepository() string {
	cmd := exec.Command("git", "remote", "-v")
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("running %#v %#v: %v", cmd.Path, cmd.Args, err)
	}
	lines := strings.Split(string(b), "\n")
	re := regexp.MustCompile(`(\S+)\s+\(fetch\)$`)
	for _, line := range lines {
		m := re.FindString(line)
		if len(m) > 0 {
			return m
		}
	}
	return "unparseable"
}

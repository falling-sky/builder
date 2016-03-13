package fileutil

import "testing"

func TestFilesInDir(t *testing.T) {
	found, err := FilesInDir("../html")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", found)
}

func FilesInDirBadDir(t *testing.T) {
	if _, err := FilesInDir("baddir"); err == nil {
		t.Fatal("failed to detect bad directory name")
	}
}

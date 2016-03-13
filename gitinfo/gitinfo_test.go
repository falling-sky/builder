package gitinfo

import (
	"testing"
)

func TestGitInfo(t *testing.T) {
	t.Logf("GetRevisionCount=%s\n", GetRevisionCount())
	t.Logf("GetProjectVersion=%s\n", GetProjectVersion())
	t.Logf("GetCleanVersion=%s\n", GetCleanVersion())
	t.Logf("GetLast=%s\n", GetLast())
}

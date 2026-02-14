package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateProject(t *testing.T) {
	base := filepath.Join(t.TempDir(), BaseDirName)
	root, err := EnsureAt(base)
	if err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	project, err := CreateProject(root, "My Book", []byte("fake-docx-data"))
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	for _, p := range []string{project.Root, project.SourcePath, project.ReportPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected path to exist %s: %v", p, err)
		}
	}
}

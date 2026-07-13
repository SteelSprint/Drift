package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLockFileExclude(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".filament")

	content := `# auto-generated

[spec]
x=abc123

[exclude]
testdata/
*.tmp

[site]
aaaa1111=def456

[state]
aaaa1111:x=abc123
`
	os.WriteFile(lockPath, []byte(content), 0644)

	lock, err := ReadLockFile(lockPath)
	if err != nil {
		t.Fatalf("ReadLockFile() error = %v", err)
	}

	if len(lock.Exclude) != 2 {
		t.Fatalf("expected 2 exclude patterns, got %d: %v", len(lock.Exclude), lock.Exclude)
	}
	found := map[string]bool{}
	for _, p := range lock.Exclude {
		found[p] = true
	}
	if !found["testdata/"] {
		t.Errorf("expected 'testdata/' in exclude patterns, got: %v", lock.Exclude)
	}
	if !found["*.tmp"] {
		t.Errorf("expected '*.tmp' in exclude patterns, got: %v", lock.Exclude)
	}
}

func TestWriteLockFilePreservesExclude(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".filament")

	lock := NewLockFile()
	lock.Spec["x"] = "abc123"
	lock.Exclude = []string{"testdata/", "*.tmp"}
	lock.Site["aaaa1111"] = "def456"
	lock.State["aaaa1111:x"] = "abc123"

	if err := WriteLockFile(lockPath, lock); err != nil {
		t.Fatalf("WriteLockFile() error = %v", err)
	}

	data, _ := os.ReadFile(lockPath)
	content := string(data)

	if !contains(content, "[exclude]") {
		t.Error("written lock file should contain [exclude] section")
	}
	if !contains(content, "testdata/") {
		t.Error("written lock file should contain 'testdata/' exclude pattern")
	}
	if !contains(content, "*.tmp") {
		t.Error("written lock file should contain '*.tmp' exclude pattern")
	}

	lock2, _ := ReadLockFile(lockPath)
	if len(lock2.Exclude) != 2 {
		t.Errorf("re-read lock file should have 2 exclude patterns, got %d", len(lock2.Exclude))
	}
}

func TestWalkPathsExcludesPatterns(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "testdata"), 0755)
	os.MkdirAll(filepath.Join(dir, "src"), 0755)

	os.WriteFile(filepath.Join(dir, "testdata", "fixture.go"), []byte("package test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.tmp"), []byte("temp\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme\n"), 0644)

	lock := NewLockFile()
	lock.Exclude = []string{"testdata/", "*.tmp"}

	files, err := WalkPathsWithExcludes([]string{dir}, lock.Exclude, dir)
	if err != nil {
		t.Fatalf("WalkPathsWithExcludes() error = %v", err)
	}

	for _, f := range files {
		rel, _ := filepath.Rel(dir, f)
		if contains(rel, "testdata") {
			t.Errorf("testdata file should be excluded, but found: %s", f)
		}
		if filepath.Ext(f) == ".tmp" {
			t.Errorf(".tmp file should be excluded, but found: %s", f)
		}
	}

	found := false
	for _, f := range files {
		if contains(f, "main.go") {
			found = true
		}
	}
	if !found {
		t.Error("src/main.go should not be excluded")
	}
}

func TestWalkPathsNoExcludes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "testdata"), 0755)
	os.WriteFile(filepath.Join(dir, "testdata", "fixture.go"), []byte("package test\n"), 0644)

	files, err := WalkPathsWithExcludes([]string{dir}, nil, dir)
	if err != nil {
		t.Fatalf("WalkPathsWithExcludes() error = %v", err)
	}

	found := false
	for _, f := range files {
		if contains(f, "fixture.go") {
			found = true
		}
	}
	if !found {
		t.Error("testdata/fixture.go should be found when no excludes are set")
	}
}

func TestSyncPreservesExclude(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".filament")

	lock := NewLockFile()
	lock.Spec["x"] = "old"
	lock.Exclude = []string{"testdata/"}
	lock.Site["aaaa1111"] = "def456"
	lock.State["aaaa1111:x"] = "old"
	WriteLockFile(lockPath, lock)

	lock2, _ := ReadLockFile(lockPath)
	lock2.Spec["x"] = "new"
	lock2.Spec["y"] = "new2"
	WriteLockFile(lockPath, lock2)

	lock3, _ := ReadLockFile(lockPath)
	if len(lock3.Exclude) != 1 || lock3.Exclude[0] != "testdata/" {
		t.Errorf("exclude section should be preserved across sync, got: %v", lock3.Exclude)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

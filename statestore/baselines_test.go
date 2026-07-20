package statestore_test

import (
	"os"
	"path/filepath"
	"testing"

	"drift/internal/fileio"
	"drift/internal/testutil"
	"drift/statestore"
)

// newTestBaselineStore returns a BaselineStore plus a live Session rooted in
// a fresh .drift/ inside a temp dir. The Session is closed via t.Cleanup.
func newTestBaselineStore(t *testing.T) (*statestore.BaselineStore, *fileio.Session) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".drift"), 0755); err != nil {
		t.Fatal(err)
	}
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatalf("fileio.Begin: %v", err)
	}
	t.Cleanup(func() { sess.Close() })
	return statestore.NewBaselineStore(), sess
}

func TestBaselineStoreWriteReadRoundTrip(t *testing.T) {
	store, sess := newTestBaselineStore(t)

	content := "line1\nline2\nline3\n"
	hash := testutil.ExpectedSha1Hex(content)

	if err := store.Write(sess, hash, content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, ok := store.Read(sess, hash)
	if !ok {
		t.Fatalf("Read returned ok=false for existing baseline")
	}
	if got != content {
		t.Fatalf("Read content mismatch: got %q, want %q", got, content)
	}
}

func TestBaselineStoreWriteDedup(t *testing.T) {
	store, sess := newTestBaselineStore(t)

	content := "same\n"
	hash := testutil.ExpectedSha1Hex(content)

	if err := store.Write(sess, hash, content); err != nil {
		t.Fatal(err)
	}
	if err := store.Write(sess, hash, content); err != nil {
		t.Fatalf("second Write failed: %v", err)
	}

	got, ok := store.Read(sess, hash)
	if !ok || got != content {
		t.Fatalf("dedup Read mismatch: ok=%v got=%q", ok, got)
	}
}

func TestBaselineStoreReadMissing(t *testing.T) {
	store, sess := newTestBaselineStore(t)

	if _, ok := store.Read(sess, "nonexistenthash"); ok {
		t.Fatalf("Read returned ok=true for missing baseline")
	}
}

// TestBaselineStoreReadNoPackfile: when baselines.bin does not exist (fresh
// project), Read returns false without erroring.
func TestBaselineStoreReadNoPackfile(t *testing.T) {
	store, sess := newTestBaselineStore(t)
	if _, ok := store.Read(sess, "anything"); ok {
		t.Fatalf("Read returned ok=true with no packfile")
	}
}

func TestBaselineStoreWriteToleratesHashMismatch(t *testing.T) {
	// The canonical hash (refs stripped) does not equal sha1(raw content)
	// for specs; Write must tolerate the mismatch.
	store, sess := newTestBaselineStore(t)

	content := "actual\n"
	declaredHash := "deadbeef" // does not match sha1(content)
	if err := store.Write(sess, declaredHash, content); err != nil {
		t.Fatalf("Write returned error for hash mismatch: %v", err)
	}
	got, ok := store.Read(sess, declaredHash)
	if !ok {
		t.Fatalf("baseline should exist after Write")
	}
	if got != content {
		t.Fatalf("Read returned %q, want %q", got, content)
	}
}

// TestBaselineStoreOverwriteExisting: Write replaces content when the hash
// matches but content differs (e.g. canonical hash collision tolerance).
func TestBaselineStoreOverwriteExisting(t *testing.T) {
	store, sess := newTestBaselineStore(t)
	hash := "fixedhash"
	if err := store.Write(sess, hash, "first"); err != nil {
		t.Fatal(err)
	}
	if err := store.Write(sess, hash, "second"); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Read(sess, hash)
	if got != "second" {
		t.Fatalf("expected overwrite to second, got %q", got)
	}
}

func TestBaselineStoreDelete(t *testing.T) {
	store, sess := newTestBaselineStore(t)
	content := "toDelete\n"
	hash := testutil.ExpectedSha1Hex(content)
	if err := store.Write(sess, hash, content); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(sess, hash); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok := store.Read(sess, hash); ok {
		t.Fatalf("Read returned ok=true after Delete")
	}
}

func TestBaselineStoreDeleteMissing(t *testing.T) {
	store, sess := newTestBaselineStore(t)
	if err := store.Delete(sess, "neverExisted"); err != nil {
		t.Fatalf("Delete missing entry should not error: %v", err)
	}
}

// TestBaselineStorePersistsAcrossSessions: a fresh Session sees what a prior
// Session wrote (proves the packfile is committed to disk, not just memory).
func TestBaselineStorePersistsAcrossSessions(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".drift"), 0755); err != nil {
		t.Fatal(err)
	}
	sess1, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	store1 := statestore.NewBaselineStore()
	if err := store1.Write(sess1, "hashA", "contentA"); err != nil {
		t.Fatal(err)
	}
	sess1.Close()

	sess2, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sess2.Close()
	store2 := statestore.NewBaselineStore()
	got, ok := store2.Read(sess2, "hashA")
	if !ok {
		t.Fatal("expected persistence across sessions, got miss")
	}
	if got != "contentA" {
		t.Fatalf("got %q, want contentA", got)
	}
}

// TestBaselineStoreManyEntriesRoundTrip: write N distinct entries and verify
// they all survive a reload (catches truncation / partial-encode bugs).
func TestBaselineStoreManyEntriesRoundTrip(t *testing.T) {
	store, sess := newTestBaselineStore(t)
	const n = 50
	want := make(map[string]string, n)
	for i := 0; i < n; i++ {
		hash := testutil.ExpectedSha1Hex(string(rune('A' + i%26)) + "x")
		content := "content-" + string(rune('A'+i%26)) + "-" + string(rune('0'+i/26))
		if err := store.Write(sess, hash, content); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
		want[hash] = content
	}
	for hash, expectedContent := range want {
		got, ok := store.Read(sess, hash)
		if !ok {
			t.Errorf("missing %s after batch write", hash)
		}
		if got != expectedContent {
			t.Errorf("content for %s: got %q want %q", hash, got, expectedContent)
		}
	}
}

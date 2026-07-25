package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"drift/internal/fileio"
)

func newFrictionSession(t *testing.T) (*fileio.Session, string) {
	t.Helper()
	dir := t.TempDir()
	driftDir := filepath.Join(dir, ".drift")
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sess.Close() })
	return sess, driftDir
}

func writeFrictionFile(t *testing.T, driftDir string, ff frictionFile) {
	t.Helper()
	data, err := json.Marshal(ff)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driftDir, frictionFileName), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFriction_PermitsWhenFileMissing(t *testing.T) {
	sess, _ := newFrictionSession(t)
	if err := checkFriction(sess); err != nil {
		t.Fatalf("expected nil for missing file, got: %v", err)
	}
}

func TestCheckFriction_PermitsWhenFileMalformed(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	if err := os.WriteFile(filepath.Join(driftDir, frictionFileName), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkFriction(sess); err != nil {
		t.Fatalf("expected nil for malformed file, got: %v", err)
	}
}

func TestCheckFriction_PermitsBelowBurst(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	now := time.Now().Unix()
	writeFrictionFile(t, driftDir, frictionFile{Resets: []int64{now - 1, now - 2}})
	if err := checkFriction(sess); err != nil {
		t.Fatalf("expected nil for 2 recent resets (< burst=3), got: %v", err)
	}
}

func TestCheckFriction_PermitsWhenStale(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	now := time.Now().Unix()
	old := now - int64((frictionWindow + 5*time.Second)/time.Second)
	writeFrictionFile(t, driftDir, frictionFile{Resets: []int64{old, old, old, old, old}})
	if err := checkFriction(sess); err != nil {
		t.Fatalf("expected nil for stale-only timestamps, got: %v", err)
	}
}

func TestCheckFriction_BlocksAtBurst(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	now := time.Now().Unix()
	writeFrictionFile(t, driftDir, frictionFile{Resets: []int64{now - 1, now - 2, now - 3}})
	if err := checkFriction(sess); err == nil {
		t.Fatalf("expected errFrictionBlocked for 3 recent resets, got nil")
	}
}

func TestCheckFriction_BlockMessageDoesNotAdvertiseOverride(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	now := time.Now().Unix()
	writeFrictionFile(t, driftDir, frictionFile{Resets: []int64{now - 1, now - 2, now - 3}})
	err := checkFriction(sess)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	for _, bad := range []string{"--dangerously-override-friction", "override", "bypass"} {
		if contains(msg, bad) {
			t.Errorf("block message advertises %q (must NOT, per R1): %s", bad, msg)
		}
	}
	if !contains(msg, "friction") {
		t.Errorf("block message should name the friction principle: %s", msg)
	}
}

func TestRecordReset_AppendsTimestamp(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	before := time.Now().Unix()
	if err := recordReset(sess); err != nil {
		t.Fatalf("recordReset: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(driftDir, frictionFileName))
	if err != nil {
		t.Fatal(err)
	}
	var ff frictionFile
	if err := json.Unmarshal(data, &ff); err != nil {
		t.Fatal(err)
	}
	if len(ff.Resets) != 1 {
		t.Fatalf("expected 1 timestamp, got %d", len(ff.Resets))
	}
	if ff.Resets[0] < before {
		t.Errorf("timestamp %d before recordReset call %d", ff.Resets[0], before)
	}
}

func TestRecordReset_PrunesOldEntries(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	now := time.Now().Unix()
	pruneCutoff := now - int64(frictionPruneAge/time.Second)
	writeFrictionFile(t, driftDir, frictionFile{Resets: []int64{
		pruneCutoff - 100, // ancient: prune
		pruneCutoff - 50,  // ancient: prune
		now - 5,           // recent: keep
	}})
	if err := recordReset(sess); err != nil {
		t.Fatalf("recordReset: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(driftDir, frictionFileName))
	if err != nil {
		t.Fatal(err)
	}
	var ff frictionFile
	if err := json.Unmarshal(data, &ff); err != nil {
		t.Fatal(err)
	}
	if len(ff.Resets) != 2 {
		t.Errorf("expected 2 timestamps after prune (1 kept + 1 new), got %d: %v", len(ff.Resets), ff.Resets)
	}
}

func TestRecordReset_RecoverFromMalformedFile(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	if err := os.WriteFile(filepath.Join(driftDir, frictionFileName), []byte("{bogus"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := recordReset(sess); err != nil {
		t.Fatalf("recordReset on malformed file: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(driftDir, frictionFileName))
	if err != nil {
		t.Fatal(err)
	}
	var ff frictionFile
	if err := json.Unmarshal(data, &ff); err != nil {
		t.Fatalf("file should be valid JSON after recordReset, got: %v", err)
	}
	if len(ff.Resets) != 1 {
		t.Errorf("expected 1 timestamp (recovered + new), got %d", len(ff.Resets))
	}
}

func TestCheckFriction_ResetsAfterExceedingThenAging(t *testing.T) {
	sess, driftDir := newFrictionSession(t)
	now := time.Now().Unix()
	writeFrictionFile(t, driftDir, frictionFile{Resets: []int64{now - 1, now - 2, now - 3}})
	if err := checkFriction(sess); err == nil {
		t.Fatal("expected block at burst")
	}
	// All timestamps now age out of the window (35s old > 30s window).
	writeFrictionFile(t, driftDir, frictionFile{Resets: []int64{
		now - 35,
		now - 36,
		now - 37,
	}})
	if err := checkFriction(sess); err != nil {
		t.Errorf("expected nil after aging out, got: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

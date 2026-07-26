package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveTheme_Level1CustomTheme: .drift/theme.xml overrides everything.
func TestResolveTheme_Level1CustomTheme(t *testing.T) {
	dir := t.TempDir()
	sess := beginSettingsSession(t, dir)
	writeCustomThemeFile(t, dir)

	// Also write user-settings — must be ignored when custom theme present.
	os.WriteFile(filepath.Join(dir, ".drift", "user-settings.xml"),
		[]byte(`<settings><theme>gruvbox</theme></settings>`), 0o644)

	theme := resolveTheme(sess)
	if theme.Name != "custom" {
		t.Fatalf("custom theme should win; got Name=%q", theme.Name)
	}
}

// TestResolveTheme_Level2UserSettings: when no custom theme, user-settings
// built-in name applies.
func TestResolveTheme_Level2UserSettings(t *testing.T) {
	dir := t.TempDir()
	sess := beginSettingsSession(t, dir)
	os.WriteFile(filepath.Join(dir, ".drift", "user-settings.xml"),
		[]byte(`<settings><theme>gruvbox</theme></settings>`), 0o644)

	theme := resolveTheme(sess)
	if theme.Name != "gruvbox" {
		t.Fatalf("user-settings theme should apply; got Name=%q", theme.Name)
	}
}

// TestResolveTheme_Level3Default: no files → DefaultTheme.
func TestResolveTheme_Level3Default(t *testing.T) {
	dir := t.TempDir()
	sess := beginSettingsSession(t, dir)

	theme := resolveTheme(sess)
	if theme.Name != DefaultTheme.Name {
		t.Fatalf("expected DefaultTheme %q, got %q", DefaultTheme.Name, theme.Name)
	}
}

// TestResolveTheme_InvalidUserSettingsName: invalid theme name in user-settings
// prints a warning to stderr and falls back to DefaultTheme.
func TestResolveTheme_InvalidUserSettingsName(t *testing.T) {
	dir := t.TempDir()
	sess := beginSettingsSession(t, dir)
	os.WriteFile(filepath.Join(dir, ".drift", "user-settings.xml"),
		[]byte(`<settings><theme>nonexistent-theme</theme></settings>`), 0o644)

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

	theme := resolveTheme(sess)
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if theme.Name != DefaultTheme.Name {
		t.Fatalf("invalid theme name should fall back to DefaultTheme %q, got %q", DefaultTheme.Name, theme.Name)
	}
	if !bytes.Contains(buf.Bytes(), []byte("warning:")) {
		t.Fatalf("expected warning on stderr, got: %q", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("nonexistent-theme")) {
		t.Fatalf("warning should name the invalid theme, got: %q", buf.String())
	}
}

// TestResolveTheme_EmptyUserSettingsFallsBack: user-settings.xml present but
// theme element empty → falls back to DefaultTheme (level 3), no warning.
func TestResolveTheme_EmptyUserSettingsFallsBack(t *testing.T) {
	dir := t.TempDir()
	sess := beginSettingsSession(t, dir)
	os.WriteFile(filepath.Join(dir, ".drift", "user-settings.xml"),
		[]byte(`<settings><theme></theme></settings>`), 0o644)

	theme := resolveTheme(sess)
	if theme.Name != DefaultTheme.Name {
		t.Fatalf("empty theme should fall back to DefaultTheme %q, got %q", DefaultTheme.Name, theme.Name)
	}
}

// writeCustomThemeFile writes a valid .drift/theme.xml with all 18 elements.
func writeCustomThemeFile(t *testing.T, dir string) {
	t.Helper()
	var elements []byte
	for _, id := range AllElementIDs {
		elements = append(elements, []byte(`<element id="`+id+`" color="red"/>`+"\n")...)
	}
	content := []byte(`<theme>` + string(elements) + `</theme>`)
	if err := os.WriteFile(filepath.Join(dir, ".drift", "theme.xml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
}

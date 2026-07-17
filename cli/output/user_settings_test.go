package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUserSettings(t *testing.T) {
	t.Run("not_exist_returns_empty", func(t *testing.T) {
		dir := t.TempDir()
		settings, err := LoadUserSettings(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if settings.Theme != "" {
			t.Errorf("Theme = %q, want empty", settings.Theme)
		}
	})

	t.Run("valid_theme", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".drift"), 0755)
		os.WriteFile(filepath.Join(dir, ".drift", "user-settings.xml"),
			[]byte(`<settings><theme>gruvbox</theme></settings>`), 0644)

		settings, err := LoadUserSettings(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if settings.Theme != "gruvbox" {
			t.Errorf("Theme = %q, want %q", settings.Theme, "gruvbox")
		}
	})

	t.Run("empty_theme_element", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".drift"), 0755)
		os.WriteFile(filepath.Join(dir, ".drift", "user-settings.xml"),
			[]byte(`<settings><theme></theme></settings>`), 0644)

		settings, err := LoadUserSettings(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if settings.Theme != "" {
			t.Errorf("Theme = %q, want empty", settings.Theme)
		}
	})

	t.Run("malformed_xml_returns_error", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".drift"), 0755)
		os.WriteFile(filepath.Join(dir, ".drift", "user-settings.xml"),
			[]byte(`<settings><theme>gruvbox`), 0644)

		_, err := LoadUserSettings(dir)
		if err == nil {
			t.Fatal("expected error for malformed XML, got nil")
		}
	})
}

func TestSaveUserSettings(t *testing.T) {
	t.Run("write_and_read_back", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".drift"), 0755)

		err := SaveUserSettings(dir, UserSettings{Theme: "nord"})
		if err != nil {
			t.Fatalf("SaveUserSettings failed: %v", err)
		}

		settings, err := LoadUserSettings(dir)
		if err != nil {
			t.Fatalf("LoadUserSettings failed: %v", err)
		}
		if settings.Theme != "nord" {
			t.Errorf("Theme = %q, want %q", settings.Theme, "nord")
		}
	})

	t.Run("overwrite_existing", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".drift"), 0755)

		SaveUserSettings(dir, UserSettings{Theme: "gruvbox"})
		SaveUserSettings(dir, UserSettings{Theme: "dracula"})

		settings, _ := LoadUserSettings(dir)
		if settings.Theme != "dracula" {
			t.Errorf("Theme = %q, want %q after overwrite", settings.Theme, "dracula")
		}
	})

	t.Run("creates_drift_dir_if_missing", func(t *testing.T) {
		dir := t.TempDir()
		// Don't create .drift/ — SaveUserSettings should create it
		err := SaveUserSettings(dir, UserSettings{Theme: "solarized-dark"})
		if err != nil {
			t.Fatalf("SaveUserSettings should create .drift/, got: %v", err)
		}
		settings, _ := LoadUserSettings(dir)
		if settings.Theme != "solarized-dark" {
			t.Errorf("Theme = %q, want %q", settings.Theme, "solarized-dark")
		}
	})
}

package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// D! id=otty range-start
// IsTerminal reports whether f is a terminal (character device). Uses stdlib
// only — no golang.org/x/term — preserving drift's zero-dependency commitment.
func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// hasNoColor checks whether the NO_COLOR environment variable is present and
// non-empty, per the https://no-color.org convention.
func hasNoColor(env []string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, "NO_COLOR=") {
			val := strings.TrimPrefix(e, "NO_COLOR=")
			if val != "" {
				return true
			}
		}
	}
	return false
}

// hasFlag reports whether flag appears in args.
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// colorModeValue returns the value of --color=X from args, or "" if not present.
func colorModeValue(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "--color=") {
			return strings.TrimPrefix(a, "--color=")
		}
	}
	return ""
}

// SelectPresenter determines the output mode based on global flags,
// environment variables, TTY status, and resolved theme. Precedence
// (highest → lowest):
//  1. --json           → JSONPresenter (theme ignored)
//  2. --no-color       → PlainPresenter (theme ignored)
//  3. --color=never    → PlainPresenter (theme ignored)
//  4. --color=always   → ColorPresenter with resolved theme
//  5. NO_COLOR env set → PlainPresenter
//  6. stdout not TTY   → PlainPresenter
//  7. default          → ColorPresenter with resolved theme
//
// Theme resolution (see output.custom_theme spec):
//  1. .drift/theme.xml (project-level, committed)
//  2. .drift/user-settings.xml (user-level, NOT committed)
//  3. DefaultTheme
//
// Theme files are read with plain file reads (os.ReadFile) — NOT via a
// fileio.Session. Theme resolution must never create .drift/ or take the
// state lock: it runs for every invocation (including bare `drift`, `drift
// help`, and unknown commands) before dispatch. Writers of these files use
// atomic rename, so a concurrent reader sees either the old or the new file —
// never a torn read.
func SelectPresenter(args []string, stdout *os.File, env []string, dir string) Presenter {
	if hasFlag(args, "--json") {
		return JSONPresenter{}
	}
	if hasFlag(args, "--no-color") {
		return PlainPresenter{}
	}
	colorAlways := false
	switch colorModeValue(args) {
	case "never":
		return PlainPresenter{}
	case "always":
		colorAlways = true
	}
	if !colorAlways {
		if hasNoColor(env) {
			return PlainPresenter{}
		}
		if !IsTerminal(stdout) {
			return PlainPresenter{}
		}
	}
	return ColorPresenter{Theme: resolveThemeFromDir(dir)}
}

// readDriftFile returns the bytes of .drift/<name>, or a not-exist error when
// the file (or the .drift/ directory itself) is absent. Plain read — no lock.
func readDriftFile(dir, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, ".drift", name))
}

// resolveThemeFromDir resolves the theme by reading .drift/theme.xml and
// .drift/user-settings.xml directly from disk. It never locks and never
// creates anything; missing files simply mean lower-precedence defaults.
func resolveThemeFromDir(dir string) Theme {
	return resolveTheme(func(name string) ([]byte, error) {
		return readDriftFile(dir, name)
	})
}

// resolveTheme returns the effective Theme using the 3-level precedence,
// reading each level through the supplied reader:
//  1. .drift/theme.xml (project-level full definition, committed)
//  2. .drift/user-settings.xml (user preference, NOT committed)
//  3. DefaultTheme
//
// If user-settings.xml contains an invalid theme name, a warning is printed
// to stderr and DefaultTheme is used.
func resolveTheme(read func(name string) ([]byte, error)) Theme {
	// 1. Project-level custom theme (all 18 elements)
	if data, err := read("theme.xml"); err == nil {
		if custom, err := ParseCustomTheme(data); err == nil {
			return custom
		}
	}
	// 2. User preference (built-in theme name)
	if data, err := read("user-settings.xml"); err == nil {
		if settings, err := ParseUserSettings(data); err == nil && settings.Theme != "" {
			if theme, ok := lookupTheme(settings.Theme); ok {
				return theme
			}
			fmt.Fprintf(os.Stderr, "warning: unknown theme %q in user-settings.xml, using default\n", settings.Theme)
		}
	}
	// 3. Default
	return DefaultTheme
}

// D! id=otty range-end

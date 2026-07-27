package commands

import (
	"fmt"
	"os"
	"path/filepath"
)

var markerSyntax = "D" + "! id=<markerid>"

// D! id=cinit range-start
func writeInitFile(dir, template string) (bool, error) {
	mainPath := filepath.Join(dir, "main.drift.xml")
	wroteMain := false
	if _, err := os.Stat(mainPath); err != nil {
		if err := os.WriteFile(mainPath, []byte(template), 0644); err != nil {
			return false, err
		}
		wroteMain = true
	}

	// Always create .drift/.gitignore so user-settings.xml and runtime
	// artifacts are never committed, even when main.drift.xml already exists.
	gitignorePath := filepath.Join(dir, ".drift", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := "# User-specific settings — not committed\nuser-settings.xml\n# Runtime lock file — not committed\nstate.lock\n# Rate-limit telemetry — not committed\nfriction.json\n"
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return wroteMain, fmt.Errorf("failed to write .drift/.gitignore: %w", err)
		}
	}
	return wroteMain, nil
}

// D! id=cinit range-end

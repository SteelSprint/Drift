package commands

import (
	"os"
	"path/filepath"
)

var markerSyntax = "D" + "! id=<markerid>"

// D! id=cinit range-start
func writeInitFile(dir, template string) error {
	mainPath := filepath.Join(dir, "main.drift.xml")
	if _, err := os.Stat(mainPath); err == nil {
		return nil
	}
	if err := os.WriteFile(mainPath, []byte(template), 0644); err != nil {
		return err
	}

	// Create .drift/.gitignore so user-settings.xml and runtime artifacts are never committed
	gitignorePath := filepath.Join(dir, ".drift", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := "# User-specific settings — not committed\nuser-settings.xml\n# Runtime lock file — not committed\nstate.lock\n# Rate-limit telemetry — not committed\nfriction.json\n"
		os.WriteFile(gitignorePath, []byte(content), 0644)
	}
	return nil
}

// D! id=cinit range-end

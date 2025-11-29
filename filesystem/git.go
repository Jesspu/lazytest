package filesystem

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// GetChangedFiles returns a list of absolute paths of files that have been modified
// or added according to git.
func GetChangedFiles(root string) ([]string, error) {
	// git status --porcelain gives us a stable, easy-to-parse output
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		// The first two characters are status codes, followed by a space, then the path
		// e.g. " M src/app.tsx" or "?? newfile.ts"
		// We care about the path, which starts at index 3
		relPath := line[3:]

		// Handle potential quotes in filename (git output behavior)
		relPath = strings.Trim(relPath, "\"")

		absPath := filepath.Join(root, relPath)
		files = append(files, absPath)
	}

	return files, nil
}

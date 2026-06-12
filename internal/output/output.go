package output

import (
	"fmt"
	"os"
	"path/filepath"
)

func WritePages(outDir string, pages []string) error {
	dir := filepath.Join(outDir, "pages")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create pages dir: %w", err)
	}

	for i, content := range pages {
		name := fmt.Sprintf("page_%03d.md", i+1)
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	return nil
}

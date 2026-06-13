package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func pagePath(outDir string, index int) string {
	name := fmt.Sprintf("page_%03d.md", index+1)
	return filepath.Join(outDir, "pages", name)
}

func pagesDir(outDir string) string {
	return filepath.Join(outDir, "pages")
}

func WritePage(outDir string, index int, content string) error {
	dir := pagesDir(outDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create pages dir: %w", err)
	}
	path := pagePath(outDir, index)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write page %d: %w", index+1, err)
	}
	return nil
}

func HasPage(outDir string, index int) bool {
	_, err := os.Stat(pagePath(outDir, index))
	return err == nil
}

func LastPageIndex(outDir string) int {
	dir := pagesDir(outDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return -1
	}
	var nums []int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		s := strings.TrimSuffix(e.Name(), ".md")
		s = strings.TrimPrefix(s, "page_")
		n, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return -1
	}
	sort.Ints(nums)
	return nums[len(nums)-1] - 1 // 0-indexed
}

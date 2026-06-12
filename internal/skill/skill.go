package skill

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Skill struct {
	Name string
	Path string
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".scansage", "skills")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func List() ([]Skill, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var skills []Skill
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if ext := filepath.Ext(name); ext == ".exe" {
			name = strings.TrimSuffix(name, ext)
		}
		skills = append(skills, Skill{Name: name, Path: filepath.Join(dir, e.Name())})
	}
	return skills, nil
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type release struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

func Install(repo string) error {
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "github.com/")
	parts := strings.Split(repo, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid repo format: use <user>/<repo>")
	}
	owner, repoName := parts[0], parts[1]

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api %d: %s", resp.StatusCode, string(body))
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return fmt.Errorf("decode release: %w", err)
	}

	goarch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		goarch += ".exe"
	}

	var asset *releaseAsset
	for _, a := range rel.Assets {
		if strings.Contains(a.Name, goarch) {
			asset = &a
			break
		}
	}
	if asset == nil {
		return fmt.Errorf("no asset found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, rel.TagName)
	}

	skillDir, err := Dir()
	if err != nil {
		return err
	}

	binName := repoName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	dest := filepath.Join(skillDir, binName)

	dlResp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer dlResp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, dlResp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if err := os.Chmod(dest, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	fmt.Printf("installed skill %s → %s\n", repoName, dest)
	return nil
}

func Run(name, dir string) error {
	skillDir, err := Dir()
	if err != nil {
		return err
	}

	binName := name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	path := filepath.Join(skillDir, binName)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("skill %q not found (use 'scansage skill install' first)", name)
	}

	// The skill receives the output directory as argument.
	// Convention: skills read pages/*.md from --dir and produce abstract.md.
	cmd := exec.Command(path, "--dir", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

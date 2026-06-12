# ScanSage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build `scansage` CLI: PDF → PNG → GLM-OCR (via llama.cpp) → raw markdown per page

**Architecture:** Single Go binary, three internal packages (render/ocr/output), main.go wires them with urfave/cli.v3

**Tech Stack:** Go 1.26.4, `github.com/gen2brain/go-fitz` (MuPDF), `github.com/urfave/cli/v3`

---

## Task 1: Project scaffold + go-fitz verification

**Files:**
- Modify: `go.mod` (add go-fitz dependency)
- Create: `internal/render/render.go` (empty stub to verify compile)
- Create: `testdata/` (directory for test PDF)

- [ ] **Step 1: Create directories**

```bash
mkdir -p internal/render internal/ocr internal/output testdata
```

- [ ] **Step 2: Add go-fitz dependency and verify build**

```bash
go get github.com/gen2brain/go-fitz
go build ./...
```

Expected: Clean build, no errors.

- [ ] **Step 3: Generate a minimal test PDF for development**

Use a Go test that creates a simple PDF, or download the go-fitz testdata:

```go
// cmd/gen_test_pdf/main.go (temporary tool, not part of final build)
package main

import (
    "os"
    "github.com/jung-kurt/gofpdf"
)

func main() {
    pdf := gofpdf.New("P", "mm", "A4", "")
    for i := 1; i <= 3; i++ {
        pdf.AddPage()
        pdf.SetFont("Helvetica", "", 16)
        pdf.Cell(40, 10, "Page")
        pdf.Cell(0, 10, fmt.Sprintf("%d", i))
    }
    pdf.OutputFileAndClose("testdata/test.pdf")
}
```

Alternatively, download go-fitz's test PDF:

```bash
curl -L -o testdata/test.pdf https://github.com/gen2brain/go-fitz/raw/master/testdata/test.pdf
```

- [ ] **Step 4: Quick render smoke test**

Create a temp test to verify go-fitz works on the system:

```go
// internal/render/render_test.go
package render_test

import (
    "testing"
    "github.com/gen2brain/go-fitz"
)

func TestRenderOpen(t *testing.T) {
    doc, err := fitz.New("../testdata/test.pdf")
    if err != nil {
        t.Fatal(err)
    }
    defer doc.Close()
    if doc.NumPage() == 0 {
        t.Fatal("expected at least 1 page")
    }
    t.Logf("pages: %d", doc.NumPage())
}
```

Run: `go test ./internal/render/ -v`
Expected: PASS, prints page count.

---

## Task 2: Render package — PDF → PNG bytes

**Files:**
- Create: `internal/render/render.go`
- Create: `internal/render/render_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/render/render_test.go
package render

import (
    "testing"
)

func TestRenderPages(t *testing.T) {
    pages, err := RenderAll("../testdata/test.pdf", 300)
    if err != nil {
        t.Fatal(err)
    }
    if len(pages) == 0 {
        t.Fatal("expected at least 1 page")
    }
    for i, png := range pages {
        if len(png) == 0 {
            t.Fatalf("page %d: empty PNG bytes", i)
        }
        t.Logf("page %d: %d bytes", i+1, len(png))
    }
}
```

Run: `go test ./internal/render/ -v`
Expected: FAIL (no RenderAll function yet)

- [ ] **Step 2: Implement RenderAll**

```go
// internal/render/render.go
package render

import (
    "fmt"
    "github.com/gen2brain/go-fitz"
)

func RenderAll(path string, dpi float64) ([][]byte, error) {
    doc, err := fitz.New(path)
    if err != nil {
        return nil, fmt.Errorf("open document: %w", err)
    }
    defer doc.Close()

    n := doc.NumPage()
    pages := make([][]byte, 0, n)

    for i := 0; i < n; i++ {
        png, err := doc.ImagePNG(i, dpi)
        if err != nil {
            return nil, fmt.Errorf("page %d: %w", i, err)
        }
        pages = append(pages, png)
    }

    return pages, nil
}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `go test ./internal/render/ -v`
Expected: PASS

---

## Task 3: OCR package — llama.cpp API client

**Files:**
- Create: `internal/ocr/ocr.go`
- Create: `internal/ocr/ocr_test.go`

- [ ] **Step 1: Write the test with a mock HTTP server**

```go
// internal/ocr/ocr_test.go
package ocr

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestOCRPage(t *testing.T) {
    // Mock llama.cpp server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/v1/chat/completions" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        resp := map[string]interface{}{
            "choices": []map[string]interface{}{
                {
                    "message": map[string]interface{}{
                        "content": "# Mock OCR Result\n\nThis is page 1.",
                    },
                },
            },
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    pngBytes := []byte("fake-png-data")
    result, err := OCRPage(server.URL, pngBytes)
    if err != nil {
        t.Fatal(err)
    }
    if result != "# Mock OCR Result\n\nThis is page 1." {
        t.Fatalf("unexpected result: %s", result)
    }
}
```

Run: `go test ./internal/ocr/ -v`
Expected: FAIL (no OCRPage function)

- [ ] **Step 2: Implement OCRPage**

```go
// internal/ocr/ocr.go
package ocr

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

type chatMessage struct {
    Role    string          `json:"role"`
    Content json.RawMessage `json:"content"`
}

type contentPart struct {
    Type     string `json:"type"`
    Text     string `json:"text,omitempty"`
    ImageURL *imgURL `json:"image_url,omitempty"`
}

type imgURL struct {
    URL string `json:"url"`
}

type chatRequest struct {
    Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
}

func OCRPage(baseURL string, pngData []byte) (string, error) {
    b64 := base64.StdEncoding.EncodeToString(pngData)
    dataURL := "data:image/png;base64," + b64

    content, _ := json.Marshal([]contentPart{
        {Type: "image_url", ImageURL: &imgURL{URL: dataURL}},
        {Type: "text", Text: "OCR markdown"},
    })

    body, _ := json.Marshal(chatRequest{
        Messages: []chatMessage{
            {Role: "user", Content: content},
        },
    })

    url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
    resp, err := http.Post(url, "application/json", bytes.NewReader(body))
    if err != nil {
        return "", fmt.Errorf("http post: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        respBody, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
    }

    var chatResp chatResponse
    if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
        return "", fmt.Errorf("decode response: %w", err)
    }

    if len(chatResp.Choices) == 0 {
        return "", fmt.Errorf("no choices in response")
    }

    return chatResp.Choices[0].Message.Content, nil
}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `go test ./internal/ocr/ -v`
Expected: PASS

---

## Task 4: Output package — write page markdown files

**Files:**
- Create: `internal/output/output.go`
- Create: `internal/output/output_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/output/output_test.go
package output

import (
    "fmt"
    "os"
    "path/filepath"
    "testing"
)

func TestWritePages(t *testing.T) {
    tmpDir := t.TempDir()
    pages := []string{
        "# Page 1\n\nContent",
        "# Page 2\n\nMore content",
    }

    err := WritePages(tmpDir, pages)
    if err != nil {
        t.Fatal(err)
    }

    for i := 0; i < len(pages); i++ {
        name := fmt.Sprintf("page_%03d.md", i+1)
        path := filepath.Join(tmpDir, "pages", name)
        data, err := os.ReadFile(path)
        if err != nil {
            t.Fatal(err)
        }
        if string(data) != pages[i] {
            t.Fatalf("%s: got %q, want %q", name, string(data), pages[i])
        }
    }
}
```

Run: `go test ./internal/output/ -v`
Expected: FAIL

- [ ] **Step 2: Implement WritePages**

```go
// internal/output/output.go
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
```

- [ ] **Step 3: Run test to verify it passes**

Run: `go test ./internal/output/ -v`
Expected: PASS

---

## Task 5: Main CLI — wire everything together

**Files:**
- Create: `main.go`

- [ ] **Step 1: Write the CLI entrypoint**

```go
// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/urfave/cli/v3"

    "github.com/Qxy0happy/scansage/internal/ocr"
    "github.com/Qxy0happy/scansage/internal/output"
    "github.com/Qxy0happy/scansage/internal/render"
)

func main() {
    cmd := &cli.Command{
        Name:  "scansage",
        Usage: "PDF → PNG → GLM-OCR → raw markdown pages",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "output",
                Aliases: []string{"o"},
                Value:   ".",
                Usage:   "output directory (creates pages/ inside)",
            },
            &cli.StringFlag{
                Name:  "ocr-url",
                Value: "http://localhost:8080",
                Usage: "llama.cpp OpenAI-compatible API URL",
            },
            &cli.FloatFlag{
                Name:  "dpi",
                Value: 300,
                Usage: "PDF rendering DPI",
            },
            &cli.IntFlag{
                Name:  "concurrency",
                Value: 1,
                Usage: "number of concurrent OCR requests",
            },
        },
        Action: func(ctx context.Context, cmd *cli.Command) error {
            input := cmd.Args().First()
            if input == "" {
                return fmt.Errorf("usage: scansage <input.pdf> [-o <dir>] [--ocr-url <url>]")
            }

            outDir := cmd.String("output")
            ocrURL := cmd.String("ocr-url")
            dpi := cmd.Float("dpi")

            // Step 1: Render
            log.Printf("rendering %s at %.0f DPI ...", input, dpi)
            pages, err := render.RenderAll(input, dpi)
            if err != nil {
                return fmt.Errorf("render: %w", err)
            }
            log.Printf("rendered %d pages", len(pages))

            // Step 2: OCR each page
            results := make([]string, len(pages))
            for i, png := range pages {
                log.Printf("OCR page %d/%d ...", i+1, len(pages))
                result, err := ocr.OCRPage(ocrURL, png)
                if err != nil {
                    return fmt.Errorf("ocr page %d: %w", i+1, err)
                }
                results[i] = result
            }
            log.Printf("OCR complete for %d pages", len(pages))

            // Step 3: Write output
            if err := output.WritePages(outDir, results); err != nil {
                return fmt.Errorf("write output: %w", err)
            }
            log.Printf("output written to %s/pages/", outDir)

            return nil
        },
    }

    if err := cmd.Run(context.Background(), os.Args); err != nil {
        log.Fatal(err)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `go build -o scansage.exe .`
Expected: Compiles without errors.

- [ ] **Step 3: Quick smoke test with test PDF**

```bash
go build -o scansage.exe .
.\scansage.exe testdata/test.pdf -o .\tmp_test --ocr-url http://localhost:8080
```

Expected: With llama.cpp running, produces `tmp_test/pages/page_001.md` etc.

---

## Task 6: Verify and commit

- [ ] **Step 1: Full build**

```bash
go build ./...
```

- [ ] **Step 2: Run all tests**

```bash
go test ./internal/... -v
```

- [ ] **Step 3: Add .gitignore**

Create `.gitignore`:
```
scansage.exe
tmp_test/
```

---

## File Structure (final)

```
scansage/
├── main.go                  # CLI entrypoint, pipeline orchestration
├── internal/
│   ├── render/
│   │   └── render.go        # go-fitz PDF → []byte PNG per page
│   ├── ocr/
│   │   └── ocr.go           # llama.cpp API client
│   └── output/
│       └── output.go        # write output/pages/page_NNN.md
├── testdata/
│   └── test.pdf             # test PDF
├── docs/
│   └── superpowers/
│       ├── specs/
│       │   └── 2025-06-12-scansage-design.md
│       └── plans/
│           └── 2025-06-12-scansage-plan.md
├── go.mod
├── go.sum
└── .gitignore
```

# ScanSage

PDF → PNG → GLM-OCR → raw markdown pages.

## Install

### Prerequisites

- Go 1.21+
- CGo (enabled by default in Go)
- C compiler (gcc, clang, or MSVC)

ScanSage uses [go-fitz](https://github.com/gen2brain/go-fitz) which bundles MuPDF — no extra library installation needed.

### Install scansage

```bash
go install github.com/Qxy0happy/scansage@v1.0.0
```

The binary will be placed at `$(go env GOPATH)/bin/scansage`. Make sure that directory is in your `PATH`.

### Install llama.cpp (OCR sidecar)

```bash
# macOS (Homebrew)
brew install llama.cpp

# Windows (WinGet)
winget install llama.cpp

# Or download pre-built binary:
# https://github.com/ggerganov/llama.cpp/releases
```

### Download GLM-OCR model

```bash
# llama.cpp will auto-download on first run:
llama-server -hf ggml-org/GLM-OCR-GGUF

# Or download manually from Hugging Face:
# https://huggingface.co/ggml-org/GLM-OCR-GGUF
```

## Usage

Start llama.cpp server in one terminal:

```bash
llama-server -hf ggml-org/GLM-OCR-GGUF
```

Run scansage in another:

```bash
scansage input.pdf
scansage input.pdf -o ./output
scansage input.pdf --ocr-url http://192.168.1.100:8080
```

Output: `output/pages/page_NNN.md` (one file per page, raw OCR markdown).

## How it works

1. **Render** — go-fitz renders each PDF page as PNG (in memory, 300 DPI)
2. **OCR** — sends each PNG to llama.cpp GLM-OCR via OpenAI-compatible API
3. **Output** — writes raw markdown to `pages/page_NNN.md`

Post-processing (LLM cleanup, summarization, layout) is done via skills.

## Skills

Skills are standalone executables that process the OCR output. They live in `~/.scansage/skills/`.

```bash
# Install a skill from GitHub releases
scansage skill install <user/repo>

# List installed skills
scansage skill list

# Run a skill on output directory
scansage skill run <name> -d ./output
```

Skills receive `--dir <path>` pointing to the output directory and can read `pages/*.md`. This keeps post-processing (refine, abstract, layout) decoupled from the OCR pipeline.

## License

AGPL-3.0

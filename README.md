# Manuscript Health Dashboard (MHD)

Local-first manuscript analysis desktop app (Go + Wails + React) with:
- Book ingestion for `.docx` and `.pdf`.
- Chapter-aware analysis pipeline.
- Health/forensics, structure, market, and language analysis tabs.
- Character dictionary and chapter-level context.
- Local service lifecycle management for `ollama` and `LanguageTool`.

## Architecture

- Root Go modules: `internal/*` for ingest, chunking, timeline, forensics, workspace, pipeline.
- Desktop backend: `desktop/backend/*` task-specific modules.
- Desktop shell: `desktop/app.go`, `desktop/service_manager.go`, `desktop/main.go`.
- Frontend: `desktop/frontend` (React + Vite).

## Workspace Output

Per analyzed manuscript, output is written under:
- `~/ManuscriptHealth/projects/{book_hash}/source.{docx|pdf}`
- `~/ManuscriptHealth/projects/{book_hash}/report.json`

`report.json` includes top-level summary fields and rich `analysis` payload:
- `language`
- `genre_scores`
- `genre_provider`
- `genre_reasoning`
- `chapter_metrics` (including `genreProvider` and `genreReasoning` per chapter)
- `chapter_summaries`
- `character_dictionary`
- `timeline`
- `beats`
- `health_issues`
- `run_stats`

## Prerequisites

- Go (current project tested with modern Go versions).
- Node/npm for frontend build.
- Wails CLI for packaging/dev UI integration.
- Ollama for high-quality local LLM analysis.
- Optional: LanguageTool binary or `LANGUAGETOOL_JAR` + Java.

## Ollama Setup (High Quality Mode)

Install and run locally:

```bash
brew install ollama
brew services start ollama
ollama pull llama3.1:8b
```

Verify:

```bash
curl -I http://localhost:11434/api/tags
ollama list
```

Recommended env for tests/runs:

```bash
export OLLAMA_LANGUAGE_MODEL=llama3.1:8b
export OLLAMA_GENRE_MODEL=llama3.1:8b
```

## Run

Quick checks:

```bash
go test ./...
go run ./cmd/mhd
```

Desktop:

```bash
cd desktop
go test ./...
```

Frontend:

```bash
cd desktop/frontend
npm install
npm run build
```

Wails app:

```bash
cd desktop
wails dev
# or
wails build
```

## E2E Scripts

- Full e2e:
  - `./scripts/run_full_e2e_test.sh`
- Full e2e + saved artifacts:
  - `./scripts/run_e2e_with_artifacts.sh`

Artifacts are saved to:
- `desktop/test-artifacts/e2e/`

### Timeout

LLM-backed chapter classification on large books is slow by design (quality over speed).  
`scripts/run_full_e2e_test.sh` uses:
- `GO_TEST_TIMEOUT` (default `45m`)

Override if needed:

```bash
GO_TEST_TIMEOUT=60m ./scripts/run_e2e_with_artifacts.sh
```

## Strict LLM Genre Tests

Integration tests enforce that genre comes from Ollama, not heuristic fallback:
- `analysis.genre_provider` must start with `ollama:`
- every `analysis.chapter_metrics[i].genreProvider` must start with `ollama:`

If Ollama is down, model missing, or malformed model output is returned, tests fail.

## Service Lifecycle

The desktop server manages dependency lifecycle:
- Attempts to start/connect `LanguageTool`.
- Attempts to start/connect `ollama`.
- Pulls required models.
- Emits service traces and diagnostics to UI.
- Stops managed processes on desktop app shutdown.

## Troubleshooting

- `genreProvider` is `heuristic` in artifacts:
  - Ensure Ollama is running and reachable on `http://localhost:11434`.
  - Ensure model exists: `ollama list`.
  - Re-run with `OLLAMA_LANGUAGE_MODEL` and `OLLAMA_GENRE_MODEL` set.

- LanguageTool unavailable:
  - Install `languagetool` binary, or set `LANGUAGETOOL_JAR` and ensure Java exists.

- macOS linker `UTType` errors in direct Go build:
  - Use `CGO_LDFLAGS='-framework UniformTypeIdentifiers'`.

## Key Paths

- `cmd/mhd/main.go`
- `internal/ingest`
- `internal/chunk`
- `internal/forensics`
- `internal/slop`
- `internal/timeline`
- `internal/workspace`
- `desktop/app.go`
- `desktop/service_manager.go`
- `desktop/backend/analyzer.go`
- `desktop/backend/genre_analysis.go`
- `desktop/books_analysis_integration_test.go`
- `scripts/run_full_e2e_test.sh`

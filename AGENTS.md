# lazyam - Agent Instructions

## Basics

- **Go 1.23.1** | `go test ./...` | no lint/typecheck/formatter configured
- Module: `github.com/Galdoba/lazyam`
- CLI entry: `cmd/lazyam/main.go` (urfave/cli/v3)

## Runtime dependencies (not in go.mod)

`ffmpeg`/`fflite`, `ffprobe`, `tgnotify` — required at runtime for transcoding and notifications.

## Architecture (key paths)

- `internal/action/run.go` — main 6-stage loop: CheckLock → ReadCache → UpdateCache → ProjectProcessing → TrailerProcessing → Sleep
- `internal/task/` — per-directory task state machine (SyncMeta → ScanSources → InterlaceCheck → Transcode)
- `internal/projectdata/` — project metadata search/caching
- `pkg/scriptkit/` — shell script template engine (`|=var=|` placeholders), templates: `Amedia1`, `Amedia2`, `Amedia2S`, `AmediaTrailer`, `ScanInterlace`
- `pkg/ump/` — ffprobe JSON wrapper (MediaProfile)
- `pkg/translit/` — Russian Cyrillic → Latin transliteration
- `pkg/error/` — `LazyError` with `IsExpected()` distinction (expected = recoverable, unexpected = abort)
- `internal/appmodule/` — config loading (TOML, XDG paths), logging bootstrap

## Config

TOML at XDG paths. Defaults point to Windows UNC paths (`//192.31.31.4/buffer/IN/...`).
See `internal/appmodule/config/config.go:49` for defaults.

## Testing

Tests exist but are **incomplete stubs** — they use `fmt.Println` instead of `t.Errorf`. Do not assume test coverage is meaningful.

```
go test ./...
```

## Gotchas

- **`internal/analitycs/`** — misspelled package name (with "it"), used throughout. Do not "fix" the spelling without updating all references.
- **Translit panics** on unknown characters in DEBUG mode (`pkg/translit/panic.txt`).
- **Path conversion** (`internal/action/run.go:334`) hardcodes Linux path mapping: `//192.168.31.4/buffer/IN` → `/home/pemaltynov/IN`.
- **Interlace detection** (`internal/task/process.go:242`) — regex expects specific ffprobe idet output format; inconclusive scans produce errors.
- **Sport content** filters out non-Russian audio tracks (`internal/action/run.go:343`).
- **Cache**: deleted on startup unless `--keep-cache` flag is set. Corrupted cache = unexpected error (abort).
- **Trailer processing** detects `*_a_treyler.mp4` files in output directory.

## Sources for deeper context

- `report.md` — full architecture and workflow documentation
- `dataflow.html` — visual data flow diagrams
- `testing/` — experimental/test harness code

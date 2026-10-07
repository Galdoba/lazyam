# lazyam - Project Report

## What is this project?

**lazyam** is an automated media content transcoding pipeline tool written in Go. It is designed for a professional media processing workflow (specifically for "AMedia" - a Russian media distribution platform). The name stands for "lazy AMedia" - an automated, cycle-based processor for media files.

## What does it do?

lazyam continuously monitors an input directory for media content (video files with associated metadata), processes them through a multi-stage pipeline, and generates transcoded output files via generated shell scripts. It handles:

- **Media ingestion**: Detects new media directories in an input folder
- **Metadata resolution**: Looks up title, season, episode info from JSON metadata files
- **Transcoding orchestration**: Generates and manages ffmpeg/fflite-based transcoding scripts
- **Interlace detection**: Analyzes video files for interlaced content using ffprobe's idet filter
- **Trailer processing**: Special handling for trailer files (naming, transcoding)
- **Caching**: Persists project/task state to JSON cache files for resilience across restarts
- **Notifications**: Sends error/warning notifications via an external `tgnotify` CLI tool (Telegram)

## Modules, systems and main mechanics

### Core Architecture

```
cmd/lazyam/           - CLI entry point (urave/cli v3)
internal/
  appmodule/          - Application context, config loading, logging
  action/             - Main processing loop with stage-based workflow
    actionstage/      - Individual stage implementations
  task/               - Task model, metadata extraction, processing phases
  projectdata/        - Project metadata storage, search, caching
  mediasource/        - Media file analysis (wraps UMP profiles)
  declare/            - Constants (app name, file names)
  flags/              - CLI flag definitions
  analitycs/          - Statistics tracking
pkg/
  scriptkit/          - Shell script templating engine
  ump/                - Universal media profile (ffprobe wrapper)
  translit/           - Russian-to-Latin transliteration engine
  notify/             - External notification sender (tgnotify)
  error/              - Custom lazy error types (expected vs unexpected)
```

### Key Modules

**1. AppContext & Config (`internal/appmodule/`)**
- Initializes the application using `github.com/Galdoba/appcontext` (XDG paths, config management, logging)
- Loads TOML configuration with defaults pointing to network paths (`//192.168.31.4/buffer/IN/`)
- Configurable: input/output directories, metadata files, cache files, processing parameters, logging

**2. Action Runner (`internal/action/run.go`)**
- Main processing loop with 6 cycle stages:
  1. **CheckLock** - Waits for lock file to be removed from input directory
  2. **ReadCache** - Loads project and task JSON caches from disk
  3. **UpdateCache** - Refreshes project metadata from source files
  4. **ProjectProcessing** - Core transcoding pipeline for each task
  5. **TrailerProcessing** - Detects and processes trailer files
  6. **Sleep** - Dormant mode (configurable sleep between cycles)
- Cycle repeats indefinitely until shutdown

**3. Task Processing (`internal/task/`)**
- Each task represents a media directory in the input folder
- Processing phases (state machine):
  - `Phase_SyncMeta` - Collect signal files (metadata.json, lock), extract metadata via recursive JSON parsing
  - `Phase_ScanSources` - Scan all media files using ffprobe via UMP
  - `Phase_StartInterlaceCheck` - Queue interlace detection
  - `Phase_EvaluateTrancecodingProcess` - Generate and launch transcoding scripts
- Metadata lookup: searches global project cache, then local metadata file
- Fallback mode: when no metadata found, uses directory name and PRT code
- Naming: constructs INBASE/OUTBASE using transliterated titles with season/episode formatting

**4. Project Data (`internal/projectdata/`)**
- `Projects` struct: map of `AmediaProject` keyed by title+GUID
- `AmediaProject`: full metadata model (title, seasons, episodes, GUID, CMS ID, genres, etc.)
- `updateProjectData()`: Converts external metadata JSON (movies/series arrays) into unified format
- `Search*` methods: Search by GUID, filekey, season, episode, title

**5. Script Kit (`pkg/scriptkit/`)**
- Template engine for generating shell scripts
- Templates use `|=variable=|` placeholder syntax
- Predefined templates:
  - `Amedia1` - Single audio track transcoding (libx264 + ALAC)
  - `Amedia2` - Dual audio track transcoding
  - `Amedia2S` - Dual audio + subtitles (SRT) transcoding
  - `AmediaTrailer` - Trailer transcoding (upscaling, sharpening)
  - `ScanInterlace` - FFmpeg idet interlace detection script
- Template selection based on detected audio track count and subtitle presence

**6. Universal Media Profile - UMP (`pkg/ump/`)**
- Wraps `ffprobe -json` output into structured `MediaProfile`
- Validates video/audio/subtitle streams
- Detects resolution (SD/HD/4K), FPS, SAR/DAR, bitrates
- Channel layout normalization (mono/stereo/5.1)
- Generates short/long profile strings for logging

**7. Transliteration (`pkg/translit/`)**
- Russian Cyrillic to Latin transliteration engine
- Configurable: white/black lists, literals map, word separators, casing
- Used to generate filesystem-safe names from Russian titles
- Supports title case, short season/episode form (`_s01_e03`)

**8. Notifications (`pkg/notify/`)**
- Sends messages via external `tgnotify` CLI tool to Telegram
- Error types: missing metadata, missing global metadata

**9. Error Handling (`pkg/error/`)**
- `LazyError` with `IsExpected()` distinction
- Expected errors (e.g., missing cache file) vs unexpected errors (e.g., disk I/O failure)

### Configuration (`internal/appmodule/config/`)

```toml
[declarations]
input_root_directory = "//192.168.31.4/buffer/IN/@AMEDIA_IN/"
output_directory = "//192.168.31.4/buffer/IN/"
metadata_files = ["//192.168.31.4/buffer/IN/@AMEDIA_IN/metadata.json"]

[processing]
interlace_threshold = 0.5        # 0-1.0 ratio
dormant_mode = 30                # seconds between cycles
cycle_lock_autoremove = 9        # auto-remove stale cycle lock
project_lock_autoremove = 27     # auto-remove stale project lock

[logging]
enabled = true
file_rotation = "none"
console_level = "trace"
```

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/urfave/cli/v3` | CLI framework (v3.4.1) |
| `github.com/Galdoba/appcontext` | Custom app context (config, logging, XDG paths) |
| `github.com/Galdoba/devtools` | Custom devtools (CLI command execution) |
| `github.com/bombsimon/jtd-infer-go` | JSON type inference (indirect) |
| `github.com/gookit/color` | Terminal color output (indirect) |
| `github.com/jsontypedef/json-typedef-go` | JSON type definitions (indirect) |
| `github.com/pelletier/go-toml/v2` | TOML config parsing (indirect) |
| `github.com/xo/terminfo` | Terminal info (indirect) |
| `golang.org/x/sys` | OS syscalls (indirect) |

**External tools required at runtime:** `ffprobe`, `ffmpeg`/`fflite`, `tgnotify`

## Main Workflow

```
START
  |
  v
[Init] Load config, create AppContext, start logging
  |
  v
[Cycle 1..N]
  |
  +--> [CheckLock]     Wait for lock file removal from input dir
  |                      (with auto-remove after timeout)
  |
  +--> [ReadCache]     Load project cache + task cache from JSON files
  |                      (create if missing, skip if --keep-cache flag)
  |
  +--> [UpdateCache]   Refresh project metadata from source JSON files
  |
  +--> [ProjectProcessing]
  |     For each task directory:
  |       1. SyncMeta: Read metadata.json, resolve title/GUID/season/episode
  |       2. ScanSources: ffprobe each media file, build media profile
  |       3. InterlaceCheck: Run idet filter, analyze interlace ratio
  |       4. Transcode: Select template (1/2/2S audio tracks), generate .sh script
  |                      Move sources to _IN_PROGRESS, create ready file
  |
  +--> [TrailerProcessing]
  |     Scan output dir for *_a_teka.mp4 files, generate transcoding scripts
  |
  +--> [Sleep]         Dormant mode for configured seconds
  |
  v
[Repeat cycle...]
```

## Edge Cases

| Edge Case | Handling |
|-----------|----------|
| **Lock file present** | Waits in loop; auto-removes after `cycle_lock_autoremove` seconds (default: 9s) |
| **Missing cache files** | Creates new empty cache; continues processing |
| **No metadata found** | Falls back to directory name + PRT code; sends Telegram notification |
| **Corrupted cache** | Returns unexpected error, aborts with error message |
| **No media files in task** | Task silently skipped (empty MediaFiles map) |
| **Empty ffprobe output** | Falls back to second ffprobe attempt with `-i` flag |
| **Multiple audio tracks** | Selects appropriate template (Amedia2 or Amedia2S); excludes non-Russian tracks for sport content |
| **Interlace detection inconclusive** | Proceeds without yadif deinterlacing filter |
| **FFmpeg/ffprobe failures** | Logged as warnings; transcoding script still generated (external execution) |
| **Sport content** | Filters out non-Russian audio tracks from transcoding |
| **Duplicate file suffixes** | Renames duplicates with index suffix (`AUDIOENG20_1`, `AUDIOENG20_2`) |
| **Network paths** | Hardcoded Windows UNC paths (`//192.168.31.4/...`); path conversion helper for Linux |
| **Trailer naming** | Parses season numbers, episode numbers from filename patterns; strips "_treyler", "_a_teka" suffixes |
| **Translit unknown character** | Panics (DEBUG mode); would pass through unchanged in non-DEBUG mode |
| **Cache save failure** | Logged as error but processing continues |
| **Graceful shutdown** | Logs message, exits with code 0 |

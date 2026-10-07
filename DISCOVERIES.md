# Script Generation Discoveries

## How script generation works (full trace)

### Entry point

`internal/action/run.go:237` — `Phase_EvaluateTrancecodingProcess` stage.

1. Collects video source and optional .srt from `activeTask.MediaFiles`
2. Calls `moveSources()` — renames files from task dir → `output_dir/{INBASE}_{filename}`
3. Calls `selectTemplate(t)` → returns template name + audio suffix list
4. Builds `[]scriptkit.ScriptArgument` with: `source`, `base_with_season`, `outbase`, `yadif`, `suffix_1`, optionally `suffix_2`/`srt`
5. Creates `scriptkit.New(scriptPath, WithTemplate(template), WithArgs(args...))`
6. Calls `transcodingProcess.CreateScriptFile()` — writes .sh to disk

### Template selection logic — `selectTemplate()` at `run.go:338`

```
Task.Suffixes() → []string of "AUDIO{LANG}{LAYOUT}" per audio stream
  e.g. ["AUDIORUS20", "AUDIOENG20", "AUDIORUS51"]

excludeRepetitions() → dedupes identical suffixes by appending index
  e.g. two stereo RUS → ["AUDIORUS20_0", "AUDIORUS20_1"]

Sport filter: if t.IsSport, drop all non-RUS audio tracks

Template selection by remaining count:
  1 audio → Amedia1 (1 audio + video)
  2 audio → Amedia2   (2 audio + video)  OR  Amedia2S if .srt present
  anything else → ("", ["UNK"]) — no template, broken output
```

### Template parameters per template

| Template | Args | Audio streams | Subtitles |
|---|---|---|---|
| `ScanInterlace` | `directory`, `file` | — (video only) | — |
| `Amedia1` | `source`, `base_with_season`, `outbase`, `yadif`, `suffix_1` | 1 (`[0:a:0]`) | — |
| `Amedia2` | + `suffix_2` | 2 (`[0:a:0]`, `[0:a:1]`) | — |
| `Amedia2S` | + `suffix_2`, `srt` | 2 (`[0:a:0]`, `[0:a:1]`) | copies .srt |
| `AmediaTrailer` | `source`, `outbase` | 1 (hardcoded RUS) | — |

### Shell script structure (all templates share)

```bash
#!/bin/bash
set -o nounset
set -o errexit
shopt -s extglob
shopt -s nullglob
PRIORITY=6          # 6 for regular, 8 for trailers

# Variable expansion via |=var=| placeholders
SOURCE="|=source=|"
OUTBASE="|=outbase=|"
# ...

# Hardcoded paths (not configurable)
TARGET_DIR="/mnt/pemaltynov/ROOT/EDIT/_amedia/${BASE_WITH_SEASON}"
BUFFER="/home/pemaltynov/IN"
IN_PROGRESS="${BUFFER}/_IN_PROGRESS"
DONE="${BUFFER}/_DONE"
NOTIFICATIONS="${BUFFER}/notifications"

# Create output dir, move source to IN_PROGRESS
mkdir -p ${TARGET_DIR}
mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"

# fflite command with filter_complex
fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \
  -filter_complex "[0:v:0]${YADIF}setsar=(1/1)[vidHD];[0:a:0]aresample=48000,atempo=25/(25/1)[aud1];[0:a:1]aresample=48000,atempo=25/(25/1)[aud2]" \
  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile high -g 0 ... \
  -map "[aud1]" -c:a alac -compression_level 0 ... \
  -map "[aud2]" -c:a alac -compression_level 0 ...

# Move source to DONE, create ready file, send notification
mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"
touch "${TARGET_DIR}/${READY_FILE}"
cp ${TARGET_DIR}/${OUTBASE}.ready ${NOTIFICATIONS}/${READY_FILE}
mv "$0" /home/pemaltynov/IN/_DONE/bash/
```

### The `|=var=|` template engine (`pkg/scriptkit/`)

- Placeholders: `|=variable_name=|`
- `argKeyFormat()` normalizes keys by stripping `|=...=|` wrapper, then re-wrapping
- `Render()` does `strings.ReplaceAll` for each placeholder → argument value
- `Validate()` checks all placeholders in template are present in args map
- `CreateScriptFile()` writes rendered string to disk with 0777 permissions

### Data flow: media file → audio suffix

```
ffprobe JSON → pkg/ump.MediaProfile (pkg/ump/ump.go:32)
  → mediasource.NewSourceMedia() (internal/mediasource/mediasource.go:20)
    → SourceFile{Languages: []string, Layout: []string}
      e.g. Languages=["rus","eng"], Layout=["20","20"]
  → Task.Suffixes() (internal/task/task.go:295)
    → "AUDIO" + UPPER(lang) + layout[l]
      e.g. "AUDIORUS20", "AUDIOENG20"
```

## Strengths

1. **Template engine is clean** — `|=var=|` syntax is simple, `scriptkit` package is well-structured with options pattern
2. **State machine is clear** — task phases in `internal/task/task.go:21-29`, transition logic in `run.go`
3. **Cache resilience** — `LazyError.IsExpected()` distinguishes recoverable (missing cache) from fatal (corrupted cache) errors
4. **ffprobe integration** — `pkg/ump/` provides comprehensive media analysis (resolution, fps, audio layout, codecs)
5. **Transliteration** — Russian titles → filesystem-safe names via `pkg/translit/`
6. **Interlace detection** — `idet` filter with threshold-based decision in `internal/task/process.go:212`

## Weaknesses (blocking the adaptive-audio goal)

1. **Hardcoded template count** — `selectTemplate()` only handles 1 or 2 audio tracks. 3+ audio → returns `("", ["UNK"])` → no template selected, script generation fails silently.

2. **Template names encode count** — `Amedia1`, `Amedia2`, `Amedia2S` — the number is baked into the template identifier, not a parameter. No `AmediaN` exists.

3. **Filter_complex is static** — Each template has a hardcoded number of `[0:a:N]` mappings and `[audN]` labels. `Amedia2` has `[aud1]` and `[aud2]`. There's no loop or dynamic generation for N audio streams.

4. **Argument keys are positional** — `suffix_1`, `suffix_2` — no `suffix_3`, `suffix_N`. Adding support would require creating `Amedia3`, `Amedia4` etc. templates manually.

5. **Output filename pattern encodes count** — `${OUTBASE}_HD_${AUDIO_SUFFIX_1}.m4a` — each audio gets its own output file. For N audio tracks, N output files with hardcoded names.

6. **Trailer template has same problem** — `AmediaTrailer` hardcodes `[0:a:0]` and `AUDIORUS20`. No multi-audio support at all.

7. **Sport filter is hardcoded** — `run.go:343` drops non-RUS audio for sport content. This is embedded in `selectTemplate()`, not configurable.

8. **No template registry** — Templates are package-level `var` values. Adding a new template means editing `templates.go` directly, then adding a new case or branch in `selectTemplate()`.

9. **fflite command is monolithic** — The entire `fflite` command is a single string literal in each template. Dynamic audio stream handling would require programmatically building the filter_complex and mapping strings.

10. **No parameterized template** — There's no "base template" with a loop variable or array expansion. Each template is a complete, self-contained shell script.

## What needs to change for adaptive audio

- **Template system**: Either parameterize the existing templates with N audio streams, or generate filter_complex and argument lists dynamically in Go before template rendering
- **`selectTemplate()`**: Return a list of audio suffixes regardless of count, and select a "base" template that supports N streams
- **Filter generation**: Build `-filter_complex` with `[0:a:0]...[aud1];[0:a:1]...[aud2];...;[0:a:N]...[audN]` dynamically
- **Output file generation**: Map each audio stream to its own output file with unique suffix
- **Template selection**: Decouple "template family" (amedia vs trailer) from "audio count" — the family determines video filters, the count determines audio mappings

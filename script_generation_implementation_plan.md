# Audio-Agnostic Script Generation — Implementation Plan

## Goal

Replace the hardcoded 1/2 audio-track templates (`Amedia1`, `Amedia2`, `Amedia2S`) with a single parameterized template that generates correct `fflite` commands for **any number of audio tracks** (1, 2, 3, …, N).

---

## Current Architecture (what must not break)

```
Task (internal/task/)
  └─ MediaFiles: map[string]mediasource.SourceFile
       └─ SourceFile.Languages []string, Layout []string
            └─ e.g. ["rus","eng"], ["20","20"]

Action runner (internal/action/run.go)
  └─ selectTemplate(t *Task) (string, []string)
       └─ returns: template name, audio suffix list
       └─ sport filter: drops non-RUS if t.IsSport
       └─ srt detection: switches Amedia2 → Amedia2S

Script generation
  └─ scriptkit.New(path, WithTemplate(template), WithArgs(args...))
       └─ args: source, base_with_season, outbase, yadif, suffix_1, [suffix_2], [srt]
       └─ template: "|=var=|" placeholders replaced by strings.ReplaceAll

Templates (pkg/scriptkit/templates.go)
  ├─ ScanInterlace  — interlace detection script (unchanged)
  ├─ Amedia1        — 1 audio track (REPLACE)
  ├─ Amedia2        — 2 audio tracks (REPLACE)
  ├─ Amedia2S       — 2 audio + srt (REPLACE)
  └─ AmediaTrailer  — trailer script (separate concern, not in scope)
```

**Shell script structure (identical across Amedia1/2/2S):**

```
#!/bin/bash — strict mode, PRIORITY=6
# Variable assignment (|=var=| → runtime values)
# Path setup (hardcoded: /mnt/pemaltynov/…, /home/pemaltynov/IN)
# mkdir + mv source → _IN_PROGRESS
# fflite command with -filter_complex and -map entries
# mv source → _DONE, create .ready, notify, move script to _DONE/bash/
```

The only difference between `Amedia1`, `Amedia2`, `Amedia2S` is:
1. **Number of `[0:a:N]` filter_complex entries** (audio resampling/labeling)
2. **Number of `-map "[audN]"` output file entries**
3. **SRT handling** (Amedia2S copies `.srt` file, 2S template has `|=srt=|` arg)

---

## Design Decisions

### 1. Template Strategy: Hybrid (static base + dynamic injection)

Keep the shell script as a template file but make the **audio-dependent portion** a single injectable argument. This avoids:
- Duplicating the entire shell script for each audio count
- Changing the `|=var=|` engine (no loop syntax needed)
- Breaking existing template infrastructure

**How it works:**
- One new template: `AmediaGeneric` with a `|=audio_section=|` placeholder
- Go code builds the filter_complex audio mappings and output map lines
- The result is a single multi-line string injected as one argument

### 2. Audio track data model

Add a lightweight struct to represent a single audio track for script generation:

```go
// pkg/scriptkit/audio_track.go (new file)

type AudioTrack struct {
    StreamIndex int       // 0-based index within the file: 0:a:0, 0:a:1, …
    Language    string    // "rus", "eng", etc.
    Layout      string    // "20" (stereo), "51" (5.1)
    Bitrate     string    // from ffprobe, e.g. "317"
    Suffix      string    // formatted label for output filename, e.g. "AUDIORUS20"
}
```

The `Suffix` field is the display label used in output filenames (`${OUTBASE}_HD_AUDIORUS20.m4a`).

### 3. Naming convention for output files

Keep the existing format: `${OUTBASE}_HD_${SUFFIX}.m4a` where `SUFFIX = AUDIO{LANG}{LAYOUT}`.

This is already used and understood by downstream systems. No change needed.

### 4. SRT handling

Subtitles are **not** audio tracks. Keep the srt-boolean logic but decouple it from the audio count:

```
HasSRT = any MediaFile has .srt extension
AudioCount = count of non-SRT MediaFiles with audio streams
```

Template selection:
- `AmediaGeneric` (always, for any audio count) + srt flag → the script includes srt copy if needed
- Remove `Amedia2S` distinction — srt handling becomes a conditional within the generic template

### 5. Backward compatibility

- Keep `Amedia1`, `Amedia2`, `Amedia2S` as aliases to `AmediaGeneric` for any code that references them by name
- Old cache files and scripts are unaffected (they're already written to disk)
- Only **new** script generation uses the new path

---

## Implementation Steps

### Step 1: Add `AudioTrack` struct and `Suffix()` formatter

**File:** `pkg/scriptkit/audio_track.go` (new)

```go
package scriptkit

type AudioTrack struct {
    StreamIndex int
    Language    string
    Layout      string
    Bitrate     string
    Suffix      string
}

func MakeAudioTrack(lang, layout string, index int) AudioTrack {
    suffix := "AUDIO" + lang + layout
    return AudioTrack{
        StreamIndex: index,
        Language:    lang,
        Layout:      layout,
        Suffix:      suffix,
    }
}
```

### Step 2: Add `GenerateAudioSection()` to build filter_complex + maps

**File:** `pkg/scriptkit/audio_track.go` (append)

```go
// GenerateAudioSection builds the fflite audio portion for N tracks.
// Returns two strings:
//   1. filter_complex segment (appended after video section)
//   2. output map lines (one per audio track)
//
// Example for 2 tracks (rus/stereo, eng/stereo):
//   filter: "[0:a:0]aresample=48000,atempo=25/(25/1)[aud1];[0:a:1]aresample=48000,atempo=25/(25/1)[aud2]"
//   maps:   -map "[aud1]" -c:a alac ... "${TARGET_DIR}/${OUTBASE}_HD_AUDIORUS20.m4a" \
//           -map "[aud2]" -c:a alac ... "${TARGET_DIR}/${OUTBASE}_HD_AUDIOENG20.m4a"
func GenerateAudioSection(tracks []AudioTrack) (filterComplex string, outputMaps string) {
    if len(tracks) == 0 {
        return "", ""
    }

    var fcParts []string   // filter_complex parts
    var mapLines []string  // output file lines

    for i, t := range tracks {
        audLabel := fmt.Sprintf("aud%d", i+1)

        // Audio resampling filter for this track
        fcParts = append(fcParts,
            fmt.Sprintf("[0:a:%d]aresample=48000,atempo=25/(25/1)[%s]", t.StreamIndex, audLabel))

        // Output file mapping for this track
        mapLines = append(mapLines,
            fmt.Sprintf(`  -map "[%s]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD_%s.m4a"`,
                audLabel, t.Suffix))
    }

    filterComplex = strings.Join(fcParts, ";")
    outputMaps = strings.Join(mapLines, " \\\n")

    return filterComplex, outputMaps
}
```

### Step 3: Create `AmediaGeneric` template

**File:** `pkg/scriptkit/templates.go` (append)

The template uses two dynamic placeholders:
- `|=audio_filter=|` — the audio filter_complex segment
- `|=audio_outputs=|` — the audio output map lines
- `|=srt_section=|` — optional SRT copy block (empty if no subtitles)

```go
var AmediaGeneric = strings.Join([]string{
    `#!/bin/bash`,
    `set -o nounset`,
    `set -o errexit`,
    `shopt -s extglob`,
    `shopt -s nullglob`,
    `PRIORITY=6`,
    ``,
    `SOURCE="|=source=|"`,
    `BASE_WITH_SEASON="|=base_with_season=|"`,
    `OUTBASE="|=outbase=|"`,
    `YADIF="|=yadif=|"`,
    ``,
    `TARGET_DIR="/mnt/pemaltynov/ROOT/EDIT/_amedia/${BASE_WITH_SEASON}"`,
    `BUFFER="/home/pemaltynov/IN"`,
    `IN_PROGRESS="${BUFFER}/_IN_PROGRESS"`,
    `DONE="${BUFFER}/_DONE"`,
    `NOTIFICATIONS="${BUFFER}/notifications"`,
    `READY_FILE="${OUTBASE}.ready"`,
    ``,
    `mkdir -p ${TARGET_DIR}`,
    `mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"`,
    `|=srt_move=|`,
    ``,
    `fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
    `  -filter_complex "[0:v:0]${YADIF}setsar=(1/1)[vidHD];|=audio_filter=|" \`,
    `  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD.mp4" \`,
    `|=audio_outputs=|`,
    `|=srt_section=|`,
    `mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"`,
    `|=srt_done=|`,
    `touch "${TARGET_DIR}/${READY_FILE}" && printf ${TARGET_DIR}/${READY_FILE} >> ${TARGET_DIR}/${READY_FILE}`,
    `cp ${TARGET_DIR}/${OUTBASE}.ready ${NOTIFICATIONS}/${READY_FILE}`,
    `mv "$0" /home/pemaltynov/IN/_DONE/bash/`,
}, "\n")
```

**Placeholder explanation:**

| Placeholder | Injected value when | Injected value when |
|---|---|---|
| `|=audio_filter=|` | 2+ audio tracks | 1 audio track (`[0:a:0]aresample=48000,atempo=25/(25/1)[aud1]`) |
| `|=audio_outputs=|` | 2+ audio tracks | 1 audio output line |
| `|=srt_move=|` | SRT file present | empty string |
| `|=srt_section=|` | SRT file present | `cp "${IN_PROGRESS}/${SRT}" "${TARGET_DIR}/${OUTBASE}.srt"\nmv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"\n` |
| `|=srt_done=|` | SRT file present | `mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"\n` |

Wait — this is getting complex with too many conditionals. Let me simplify.

**Better approach:** Use a single `|=dynamic_content=|` placeholder that contains the entire audio section (filter + outputs) and the SRT section combined. The Go code builds the full dynamic block.

Revised template:

```go
var AmediaGeneric = strings.Join([]string{
    `#!/bin/bash`,
    `set -o nounset`,
    `set -o errexit`,
    `shopt -s extglob`,
    `shopt -s nullglob`,
    `PRIORITY=6`,
    ``,
    `SOURCE="|=source=|"`,
    `BASE_WITH_SEASON="|=base_with_season=|"`,
    `OUTBASE="|=outbase=|"`,
    `YADIF="|=yadif=|"`,
    ``,
    `TARGET_DIR="/mnt/pemaltynov/ROOT/EDIT/_amedia/${BASE_WITH_SEASON}"`,
    `BUFFER="/home/pemaltynov/IN"`,
    `IN_PROGRESS="${BUFFER}/_IN_PROGRESS"`,
    `DONE="${BUFFER}/_DONE"`,
    `NOTIFICATIONS="${BUFFER}/notifications"`,
    `READY_FILE="${OUTBASE}.ready"`,
    ``,
    `mkdir -p ${TARGET_DIR}`,
    `mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"`,
    `|=srt_move=|`,
    ``,
    `fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
    `  -filter_complex "[0:v:0]${YADIF}setsar=(1/1)[vidHD];|=audio_filter=|" \`,
    `  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD.mp4" \`,
    `|=audio_outputs=|`,
    `|=srt_section=|`,
    `mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"`,
    `|=srt_done=|`,
    `touch "${TARGET_DIR}/${READY_FILE}" && printf ${TARGET_DIR}/${READY_FILE} >> ${TARGET_DIR}/${READY_FILE}`,
    `cp ${TARGET_DIR}/${OUTBASE}.ready ${NOTIFICATIONS}/${READY_FILE}`,
    `mv "$0" /home/pemaltynov/IN/_DONE/bash/`,
}, "\n")
```

With these argument conventions:
- `|=audio_filter=|` → empty string (for 1 audio, the filter is inline) or the full audio filter segment for 2+
- `|=audio_outputs=|` → the output file lines for all audio tracks
- `|=srt_move=|` → `mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"\n` or empty
- `|=srt_section=|` → subtitle copy line or empty
- `|=srt_done=|` → `mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"\n` or empty

Actually, I realize this is still messy. Let me reconsider with a cleaner approach.

**Best approach: Single dynamic content argument**

One argument `|=dynamic_section=|` that contains everything audio/SRT-dependent. The Go code assembles it.

```go
var AmediaGeneric = strings.Join([]string{
    `#!/bin/bash`,
    `set -o nounset`,
    `set -o errexit`,
    `shopt -s extglob`,
    `shopt -s nullglob`,
    `PRIORITY=6`,
    ``,
    `SOURCE="|=source=|"`,
    `BASE_WITH_SEASON="|=base_with_season=|"`,
    `OUTBASE="|=outbase=|"`,
    `YADIF="|=yadif=|"`,
    ``,
    `TARGET_DIR="/mnt/pemaltynov/ROOT/EDIT/_amedia/${BASE_WITH_SEASON}"`,
    `BUFFER="/home/pemaltynov/IN"`,
    `IN_PROGRESS="${BUFFER}/_IN_PROGRESS"`,
    `DONE="${BUFFER}/_DONE"`,
    `NOTIFICATIONS="${BUFFER}/notifications"`,
    `READY_FILE="${OUTBASE}.ready"`,
    ``,
    `mkdir -p ${TARGET_DIR}`,
    `mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"`,
    `|=dynamic_section=|`,
    ``,
    `fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
    `  -filter_complex "[0:v:0]${YADIF}setsar=(1/1)[vidHD];|=audio_filter=|" \`,
    `  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD.mp4" \`,
    `|=audio_outputs=|`,
    ``,
    `mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"`,
    `|=cleanup=|`,
    `touch "${TARGET_DIR}/${READY_FILE}" && printf ${TARGET_DIR}/${READY_FILE} >> ${TARGET_DIR}/${READY_FILE}`,
    `cp ${TARGET_DIR}/${OUTBASE}.ready ${NOTIFICATIONS}/${READY_FILE}`,
    `mv "$0" /home/pemaltynov/IN/_DONE/bash/`,
}, "\n")
```

Arguments:
- `|=audio_filter=|` → audio filter_complex segment (e.g., `[0:a:0]...[aud1];[0:a:1]...[aud2]`)
- `|=audio_outputs=|` → output map lines for each audio track
- `|=cleanup=|` → SRT copy + move lines if SRT present, empty otherwise

This is clean enough. Let me finalize the plan.

---

## Revised Implementation Steps

### Step 1: Add `AudioTrack` struct

**File:** `pkg/scriptkit/audio_track.go` (new)

```go
package scriptkit

import "strings"

type AudioTrack struct {
    StreamIndex int    // 0-based: 0:a:0, 0:a:1, ...
    Language    string // "rus", "eng"
    Layout      string // "20" (stereo), "51" (5.1)
    Bitrate     string
    Suffix      string // "AUDIORUS20"
}

func NewAudioTrack(lang, layout string, index int) AudioTrack {
    return AudioTrack{
        StreamIndex: index,
        Language:    lang,
        Layout:      layout,
        Suffix:      "AUDIO" + lang + layout,
    }
}
```

### Step 2: Add `AudioScriptSection` builder

**File:** `pkg/scriptkit/audio_track.go` (append)

```go
// AudioScriptSection builds the audio portion of the fflite command.
// Returns (filterComplexSegment, outputFileLines).
// filterComplexSegment: semicolon-separated audio filter chains
// outputFileLines: one "-map ..." line per audio track
func AudioScriptSection(tracks []AudioTrack) (filterPart, outputLines string) {
    if len(tracks) == 0 {
        return "", ""
    }

    var filterParts []string
    var outputLinesList []string

    for i, t := range tracks {
        label := fmt.Sprintf("aud%d", i+1)
        filterParts = append(filterParts,
            fmt.Sprintf("[0:a:%d]aresample=48000,atempo=25/(25/1)[%s]", t.StreamIndex, label))
        outputLinesList = append(outputLinesList,
            fmt.Sprintf(`  -map "[%s]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD_%s.m4a"`,
                label, t.Suffix))
    }

    return strings.Join(filterParts, ";"), strings.Join(outputLinesList, " \\\n")
}
```

### Step 3: Create `AmediaGeneric` template

**File:** `pkg/scriptkit/templates.go` (append at end)

```go
var AmediaGeneric = strings.Join([]string{
    `#!/bin/bash`,
    `set -o nounset`,
    `set -o errexit`,
    `shopt -s extglob`,
    `shopt -s nullglob`,
    `PRIORITY=6`,
    ``,
    `SOURCE="|=source=|"`,
    `BASE_WITH_SEASON="|=base_with_season=|"`,
    `OUTBASE="|=outbase=|"`,
    `YADIF="|=yadif=|"`,
    ``,
    `TARGET_DIR="/mnt/pemaltynov/ROOT/EDIT/_amedia/${BASE_WITH_SEASON}"`,
    `BUFFER="/home/pemaltynov/IN"`,
    `IN_PROGRESS="${BUFFER}/_IN_PROGRESS"`,
    `DONE="${BUFFER}/_DONE"`,
    `NOTIFICATIONS="${BUFFER}/notifications"`,
    `READY_FILE="${OUTBASE}.ready"`,
    ``,
    `mkdir -p ${TARGET_DIR}`,
    `mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"`,
    `|=srt_move=|`,
    ``,
    `fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
    `  -filter_complex "[0:v:0]${YADIF}setsar=(1/1)[vidHD];|=audio_filter=|" \`,
    `  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD.mp4" \`,
    `|=audio_outputs=|`,
    `|=srt_section=|`,
    `mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"`,
    `|=srt_cleanup=|`,
    `touch "${TARGET_DIR}/${READY_FILE}" && printf ${TARGET_DIR}/${READY_FILE} >> ${TARGET_DIR}/${READY_FILE}`,
    `cp ${TARGET_DIR}/${OUTBASE}.ready ${NOTIFICATIONS}/${READY_FILE}`,
    `mv "$0" /home/pemaltynov/IN/_DONE/bash/`,
}, "\n")
```

Placeholders:

| Placeholder | Content |
|---|---|
| `|=audio_filter=|` | Audio filter_complex segment from `AudioScriptSection()` |
| `|=audio_outputs=|` | Output map lines from `AudioScriptSection()` |
| `|=srt_move=|` | `mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"\n` or empty |
| `|=srt_section=|` | `cp "${IN_PROGRESS}/${SRT}" "${TARGET_DIR}/${OUTBASE}.srt"\n` or empty |
| `|=srt_cleanup=|` | `mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"\n` or empty |

### Step 4: Refactor `selectTemplate()` → `BuildScriptArgs()`

**File:** `internal/action/run.go`

Replace:
```go
func selectTemplate(t *task.Task) (string, []string)
```

With:
```go
// BuildScriptArgs assembles script arguments for the given task.
// Returns the template name and the full argument list for scriptkit.
func BuildScriptArgs(t *task.Task, srtFile string) (*scriptkit.Script, error)
```

**Logic:**

```go
func BuildScriptArgs(t *task.Task, srtFile string) (*scriptkit.Script, error) {
    // 1. Collect audio tracks from MediaFiles
    var audioTracks []scriptkit.AudioTrack
    for _, mf := range t.MediaFiles {
        if len(mf.Languages) == 0 {
            continue // not an audio file
        }
        for i, lang := range mf.Languages {
            // Sport filter: skip non-RUS if t.IsSport
            if t.IsSport && !strings.Contains(lang, "RUS") {
                continue
            }
            layout := ""
            if i < len(mf.Layout) {
                layout = mf.Layout[i]
            }
            if layout == "" {
                continue
            }
            track := scriptkit.NewAudioTrack(lang, layout, len(audioTracks))
            audioTracks = append(audioTracks, track)
        }
    }

    // 2. Deduplicate identical suffixes (handle duplicate audio streams)
    audioTracks = dedupeTracks(audioTracks)

    // 3. Build audio section
    audioFilter, audioOutputs := scriptkit.AudioScriptSection(audioTracks)

    // 4. Build SRT sections
    var srtMove, srtSection, srtCleanup string
    if srtFile != "" {
        srtMove = fmt.Sprintf(`mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"\n`)
        srtSection = fmt.Sprintf(`cp "${IN_PROGRESS}/${SRT}" "${TARGET_DIR}/${OUTBASE}.srt"\n`)
        srtCleanup = fmt.Sprintf(`mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"\n`)
    }

    // 5. Assemble arguments
    args := []scriptkit.ScriptArgument{
        scriptkit.ScriptArg("source", t.INBASE+"_"+/* source filename */),
        scriptkit.ScriptArg("base_with_season", t.TranslitedBaseSeason()),
        scriptkit.ScriptArg("outbase", t.OUTBASE),
        scriptkit.ScriptArg("yadif", ""),
        scriptkit.ScriptArg("audio_filter", audioFilter),
        scriptkit.ScriptArg("audio_outputs", audioOutputs),
        scriptkit.ScriptArg("srt_move", srtMove),
        scriptkit.ScriptArg("srt_section", srtSection),
        scriptkit.ScriptArg("srt_cleanup", srtCleanup),
    }
    if srtFile != "" {
        args = append(args, scriptkit.ScriptArg("srt", srtFile))
    }

    // 6. Create script
    scriptPath := filepath.Join(cfg.Declarations.OutputDirectory, t.INBASE+".sh")
    return scriptkit.New(scriptPath,
        scriptkit.WithTemplate(scriptkit.AmediaGeneric),
        scriptkit.WithArgs(args...),
    ), nil
}
```

### Step 5: Update `moveSources()` call site

**File:** `internal/action/run.go`, around line 250

Current:
```go
if err := moveSources(cfg, activeTask, source, srt); err != nil { ... }
template, audioSuffixes := selectTemplate(activeTask)
```

Replace with:
```go
if err := moveSources(cfg, activeTask, source, srt); err != nil { ... }

// Build script with dynamic audio support
script, err := BuildScriptArgs(activeTask, srt)
if err != nil {
    log.Errorf("failed to build transcoding script: %v", err)
    break
}

if err := script.CreateScriptFile(); err != nil {
    log.Errorf("failed to create transcoding script: %v", err)
    break
}
log.Infof("transcoding script generated: %v", script.Path())
```

### Step 6: Remove/deprecate old templates

**File:** `pkg/scriptkit/templates.go`

Keep `Amedia1`, `Amedia2`, `Amedia2S` as deprecated aliases:

```go
// Deprecated: Use AmediaGeneric instead. Kept for backward compatibility.
var Amedia1 = AmediaGeneric
var Amedia2 = AmediaGeneric
var Amedia2S = AmediaGeneric
```

Or remove them entirely if no external code references them. Check `go ref` usage.

### Step 7: Update `mediasource.SourceFile` if needed

Current `mediasource.SourceFile` stores `Languages []string` and `Layout []string` as parallel arrays. The index alignment is: `Languages[i]` corresponds to `Layout[i]`.

This is correct but fragile. Consider adding a getter:

**File:** `internal/mediasource/mediasource.go` (append)

```go
// AudioStreams returns parallel audio stream metadata.
// Languages[i] and Layouts[i] correspond to the same audio stream.
func (sf *SourceFile) AudioStreams() (languages []string, layouts []string) {
    return sf.Languages, sf.Layout
}
```

### Step 8: Handle `StreamIndex` for audio tracks

The `StreamIndex` in `AudioTrack` must match the actual stream index in the ffprobe output (`0:a:0`, `0:a:1`, etc.).

**Problem:** `SourceFile.Languages` and `SourceFile.Layout` are populated from `ump.MediaProfile.Streams` in order of detection, but the `StreamIndex` should be the original index from the file.

**Solution:** Store the stream index in `SourceFile` or compute it when building `AudioTrack`.

Option A — extend `mediasource.SourceFile`:
```go
type SourceFile struct {
    // ... existing fields ...
    AudioStreamIndices []int `json:"audio_stream_indices,omitempty"` // parallel to Languages
}
```

Option B — compute index in `BuildScriptArgs` by scanning `Task.MediaFiles`:
```go
// Count audio streams before this file to get global stream index
globalAudioIndex := 0
for fname, mf := range t.MediaFiles {
    if fname == currentFileName {
        break
    }
    if len(mf.Languages) > 0 {
        globalAudioIndex += len(mf.Languages)
    }
}
```

Option B is simpler and doesn't require schema changes. Use Option B.

### Step 9: Handle the 1-audio-track edge case

When there's exactly 1 audio track, the filter_complex looks like:
```
[0:v:0]${YADIF}setsar=(1/1)[vidHD];[0:a:0]aresample=48000,atempo=25/(25/1)[aud1]
```

This is the same code path as 2+ tracks — `AudioScriptSection` handles it. No special case needed.

### Step 10: Update tests

**File:** `pkg/scriptkit/audio_track_test.go` (new)

```go
package scriptkit

import (
    "strings"
    "testing"
)

func TestAudioScriptSection_1Track(t *testing.T) {
    tracks := []AudioTrack{
        NewAudioTrack("rus", "20", 0),
    }
    filter, outputs := AudioScriptSection(tracks)

    expectedFilter := "[0:a:0]aresample=48000,atempo=25/(25/1)[aud1]"
    if filter != expectedFilter {
        t.Errorf("filter = %q, want %q", filter, expectedFilter)
    }

    expectedOutput := `  -map "[aud1]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD_AUDIORUS20.m4a"`
    if outputs != expectedOutput {
        t.Errorf("outputs = %q, want %q", outputs, expectedOutput)
    }
}

func TestAudioScriptSection_3Tracks(t *testing.T) {
    tracks := []AudioTrack{
        NewAudioTrack("rus", "20", 0),
        NewAudioTrack("eng", "20", 1),
        NewAudioTrack("rus", "51", 2),
    }
    filter, outputs := AudioScriptSection(tracks)

    if !strings.Contains(filter, "[0:a:0]") || !strings.Contains(filter, "[0:a:1]") || !strings.Contains(filter, "[0:a:2]") {
        t.Errorf("filter missing stream indices: %s", filter)
    }
    if !strings.Contains(outputs, "AUDIORUS20") || !strings.Contains(outputs, "AUDIOENG20") || !strings.Contains(outputs, "AUDIORUS51") {
        t.Errorf("outputs missing suffixes: %s", outputs)
    }
}

func TestAudioScriptSection_Empty(t *testing.T) {
    filter, outputs := AudioScriptSection(nil)
    if filter != "" || outputs != "" {
        t.Errorf("expected empty strings for no tracks, got filter=%q outputs=%q", filter, outputs)
    }
}
```

**File:** `internal/action/run.go` — add test for `BuildScriptArgs`

---

## File Change Summary

| File | Change |
|---|---|
| `pkg/scriptkit/audio_track.go` | **NEW** — `AudioTrack` struct + `AudioScriptSection()` |
| `pkg/scriptkit/templates.go` | Append `AmediaGeneric` template; deprecate `Amedia1/2/2S` |
| `internal/action/run.go` | Replace `selectTemplate()` + inline template selection with `BuildScriptArgs()` |
| `internal/action/run.go` | Update `Phase_EvaluateTrancecodingProcess` to use `BuildScriptArgs()` |
| `pkg/scriptkit/audio_track_test.go` | **NEW** — tests for `AudioScriptSection()` |
| `internal/action/run_test.go` | **NEW** — tests for `BuildScriptArgs()` |
| `internal/mediasource/mediasource.go` | Optional: add `AudioStreams()` helper |

---

## Risk Assessment

| Risk | Severity | Mitigation |
|---|---|---|
| Stream index mismatch (audio tracks mapped to wrong ffprobe index) | High | Compute global audio index by iterating `MediaFiles` in order; add test with 3+ audio tracks |
| Sport filter loses track info | Medium | Sport filter runs during `BuildScriptArgs()`, not during track collection; verify |
| Template placeholder collision with file content | Low | `|=var=|` is unlikely to appear in real filenames |
| Backward compatibility with old scripts | None | Old scripts are already written to disk; only new generation changes |
| Empty audio track list (no audio in file) | Low | `AudioScriptSection(nil)` returns empty strings; fflite will fail gracefully with `-i` only |
| SRT file with no audio tracks | Low | SRT handling is independent of audio; `|=srt_*=` placeholders still injected correctly |

---

## Testing Strategy

1. **Unit tests** for `AudioScriptSection()` with 0, 1, 2, 3, 5 tracks
2. **Unit tests** for `BuildScriptArgs()` verifying argument count and content
3. **Integration test**: mock a task with 3 audio tracks + srt, verify generated script contains all expected `-map` entries
4. **Regression test**: verify 1-track and 2-track cases produce identical output to old templates (byte-for-byte comparison)

---

## Out of Scope (future work)

- **Trailer template** (`AmediaTrailer`) — same hardcoded audio issue, but separate concern. Fix after main change.
- **Dynamic video filters** — video is always `[0:v:0]`, no multi-video support needed
- **Template parameterization engine** — extending `|=var=|` to support loops. Overkill for current needs.
- **FFprobe stream index caching** — compute indices on-the-fly during `BuildScriptArgs()`
- **Configurable output format** — currently hardcoded to `.mp4` (video) + `.m4a` (audio). Could be templated.
- **Configurable codec** — currently hardcoded to `libx264` + `alac`. Could be templated.

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

	expectedOutput := `  -map "[aud1]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD_AUDIO_00_RUS20.m4a"`
	if outputs != expectedOutput {
		t.Errorf("outputs = %q, want %q", outputs, expectedOutput)
	}
}

func TestAudioScriptSection_2Tracks(t *testing.T) {
	tracks := []AudioTrack{
		NewAudioTrack("rus", "20", 0),
		NewAudioTrack("eng", "20", 1),
	}
	filter, outputs := AudioScriptSection(tracks)

	expectedFilter := "[0:a:0]aresample=48000,atempo=25/(25/1)[aud1];[0:a:1]aresample=48000,atempo=25/(25/1)[aud2]"
	if filter != expectedFilter {
		t.Errorf("filter = %q, want %q", filter, expectedFilter)
	}

	if !strings.Contains(outputs, "AUDIO_00_RUS20") {
		t.Errorf("outputs missing AUDIO_00_RUS20: %s", outputs)
	}
	if !strings.Contains(outputs, "AUDIO_01_ENG20") {
		t.Errorf("outputs missing AUDIO_01_ENG20: %s", outputs)
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
	if !strings.Contains(outputs, "AUDIO_00_RUS20") || !strings.Contains(outputs, "AUDIO_01_ENG20") || !strings.Contains(outputs, "AUDIO_02_RUS51") {
		t.Errorf("outputs missing suffixes: %s", outputs)
	}
}

func TestAudioScriptSection_Empty(t *testing.T) {
	filter, outputs := AudioScriptSection(nil)
	if filter != "" || outputs != "" {
		t.Errorf("expected empty strings for no tracks, got filter=%q outputs=%q", filter, outputs)
	}

	filter, outputs = AudioScriptSection([]AudioTrack{})
	if filter != "" || outputs != "" {
		t.Errorf("expected empty strings for empty track list, got filter=%q outputs=%q", filter, outputs)
	}
}

func TestAudioScriptSection_MixedLayouts(t *testing.T) {
	tracks := []AudioTrack{
		NewAudioTrack("rus", "20", 0),
		NewAudioTrack("eng", "51", 1),
	}
	filter, outputs := AudioScriptSection(tracks)

	expectedFilter := "[0:a:0]aresample=48000,atempo=25/(25/1)[aud1];[0:a:1]aresample=48000,atempo=25/(25/1)[aud2]"
	if filter != expectedFilter {
		t.Errorf("filter = %q, want %q", filter, expectedFilter)
	}

	if !strings.Contains(outputs, "AUDIO_00_RUS20") || !strings.Contains(outputs, "AUDIO_01_ENG51") {
		t.Errorf("outputs missing expected suffixes: %s", outputs)
	}
}

func TestNewAudioTrack(t *testing.T) {
	tracks := []struct {
		lang     string
		layout   string
		index    int
		wantLang string
		wantSuffix string
	}{
		{"rus", "20", 0, "rus", "AUDIO_00_RUS20"},
		{"eng", "51", 1, "eng", "AUDIO_01_ENG51"},
		{"jpn", "20", 2, "jpn", "AUDIO_02_JPN20"},
	}

	for _, tc := range tracks {
		t.Run(tc.lang, func(t *testing.T) {
			tr := NewAudioTrack(tc.lang, tc.layout, tc.index)
			if tr.Language != tc.wantLang {
				t.Errorf("Language = %q, want %q", tr.Language, tc.wantLang)
			}
			if tr.Suffix != tc.wantSuffix {
				t.Errorf("Suffix = %q, want %q", tr.Suffix, tc.wantSuffix)
			}
			if tr.Layout != tc.layout {
				t.Errorf("Layout = %q, want %q", tr.Layout, tc.layout)
			}
			if tr.StreamIndex != tc.index {
				t.Errorf("StreamIndex = %d, want %d", tr.StreamIndex, tc.index)
			}
		})
	}
}

func TestAmediaGenericTemplate(t *testing.T) {
	// Verify AmediaGeneric contains all required placeholders
	template := AmediaGeneric

	requiredPlaceholders := []string{
		"|=source=|",
		"|=base_with_season=|",
		"|=outbase=|",
		"|=yadif=|",
		"|=audio_filter=|",
		"|=audio_outputs=|",
		"|=srt_move=|",
		"|=srt_section=|",
		"|=srt_cleanup=|",
	}

	for _, ph := range requiredPlaceholders {
		if !strings.Contains(template, ph) {
			t.Errorf("template missing placeholder %q", ph)
		}
	}

	// Verify backward compatibility aliases point to AmediaGeneric
	if Amedia1 != AmediaGeneric {
		t.Error("Amedia1 should be an alias to AmediaGeneric")
	}
	if Amedia2 != AmediaGeneric {
		t.Error("Amedia2 should be an alias to AmediaGeneric")
	}
	if Amedia2S != AmediaGeneric {
		t.Error("Amedia2S should be an alias to AmediaGeneric")
	}
}

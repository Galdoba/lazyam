package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Galdoba/lazyam/internal/mediasource"
	"github.com/Galdoba/lazyam/internal/projectdata"
	"github.com/Galdoba/lazyam/pkg/translit"
	"github.com/Galdoba/lazyam/pkg/ump"
)

const (
	Phase_SyncMeta = iota
	Phase_ScanSources
	Phase_StartInterlaceCheck
	Phase_EvaluateInterlaceCheckResult
	Phase_StartTrancecoding
	Phase_EvaluateTrancecodingProcess
	Phase_WaitTranceCodingResult
	Phase_CleanData
)

func (t *Task) FillMetatada(prj *projectdata.Projects) error {
	t.PRT = strings.TrimPrefix(getPRT(t.Directory), "_")
	source := taskMeta{}
	t.PRT = getPRT(t.Directory)
	t.AmediaFileKey = ""
	project := projectdata.AmediaProject{}

	if meta, ok := t.SignalFiles["metadata"]; ok {
		data, err := os.ReadFile(meta)
		if err != nil {
			return fmt.Errorf("failed to read: %v", err)
		}
		if err := json.Unmarshal(data, &source); err != nil {
			return fmt.Errorf("failed to unmarshal: %v", err)
		}
		project = prj.SearchByGUID(source.GUID)
		t.AmediaGUID = source.GUID
		if source.GUID != project.GUID {
			t.Type = "SER"
			t.Season, t.Episode = project.SeasonEpisode(t.AmediaGUID)
		}
		if source.File.Serid != "" {
			t.AmediaFileKey = source.File.Serid
		}
	} else {
		project, t.AmediaFileKey = prj.SearchByAmediaFileKey(t.Directory)
	}
	if project.Name() != "" {
		t.AmediaTitleRus = project.RusTitle
		t.AmediaTitleOri = project.OriginalTitle
		t.OUTBASE = constructOutbase(t)
		t.INBASE = constructInbase(t)
	} else {
		t.OUTBASE = filepath.Base(t.Directory)
		t.INBASE = filepath.Base(t.Directory)
	}

	return nil
}

func (t *Task) ScanSources() error {
	fi, err := os.ReadDir(t.Directory)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}
fileLoop:
	for _, f := range fi {
		if f.IsDir() {
			continue
		}
		path := joinPath(t.Directory, f.Name())
		for _, signal := range t.SignalFiles {
			if path == signal {
				continue fileLoop
			}
		}
		mp := ump.NewProfile()
		if err := mp.ConsumeFile(path); err != nil {

			return fmt.Errorf("failed to scan media: %v", err)
		}
		t.MediaFiles[path] = mediasource.NewSourceMedia(mp)
	}
	return nil
}

func (t *Task) VideoSourceName() string {
	for _, file := range t.MediaFiles {
		if file.Type != "SOURCE" {
			continue
		}
		return file.Name
	}
	return ""
}

func (t *Task) AssesInterlaceReport() error {
	fi, err := os.ReadDir(t.Directory)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}
	for _, f := range fi {
		if !strings.HasSuffix(f.Name(), ".idet") {
			continue
		}
		idetData, err := parseIdet(filepath.Join(t.Directory, f.Name()))
		if err != nil {
			return fmt.Errorf("failed to parse idet: %v", err)
		}
		t.InderlaceScanned = true
		if idetData.interlaceScore < 95 {
			t.InterlaceDetected = true
		}
		return nil
	}
	return fmt.Errorf("scan not complete")
}

type idetScan struct {
	frameCount     map[int]int
	interlaceScore int
	err            error
}

func parseIdet(path string) (idetScan, error) {
	is := idetScan{}
	is.frameCount = make(map[int]int)
	data, err := os.ReadFile(path)
	if err != nil {
		return is, err
	}
	if len(data) < 20 {
		return is, fmt.Errorf("scan not finished 1")
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		numbers := strings.Split(line, " ")
		for _, num := range numbers {
			if val, err := strconv.Atoi(num); err == nil {
				is.frameCount[len(is.frameCount)] = val
			}
		}
	}
	total := is.frameCount[7] + is.frameCount[8] + is.frameCount[9] + is.frameCount[10]
	if total == 0 {
		return is, fmt.Errorf("scan not finished 2")
	}
	is.interlaceScore = (is.frameCount[9] * 1000) / total
	return is, nil
}

func (t *Task) TranslitedBase() string {
	base := ""
	if t.AmediaTitleRus != "" {
		base = translit.String(t.AmediaTitleRus, translit.RegisterLow())
	} else {
		base = translit.String(t.AmediaTitleOri, translit.RegisterLow())
	}
	return toTitle(base)
}

func (t *Task) TranslitedBaseSeason() string {
	out := ""
	if t.Season < 1 {
		return ""
	}
	if t.Season > 0 {
		out += "_s" + numToStr(t.Season)
	}
	return t.TranslitedBase() + out
}

func (t *Task) Suffixes() []string {
	suf := []string{}
	for _, v := range t.MediaFiles {
		if len(v.Languages) == 0 {
			continue
		}
		for _, lang := range v.Languages {
			suf = append(suf, "AUDIO"+strings.ToUpper(lang)+v.Layout[0])
		}
	}
	return suf
}

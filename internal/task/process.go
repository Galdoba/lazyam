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
	"github.com/Galdoba/lazyam/pkg/ump"
)

const (
	Phase_SyncMeta = iota
	Phase_ScanSources
	Phase_StartInterlaceCheck
	Phase_EvaluateInterlaceCheckResult
	Phase_StartTrancecoding
	Phase_EvaluateTrancecodingProcess
	Phase_CleanData
)

func (t *Task) FillMetatada(prj *projectdata.Projects) error {
	// files, err := os.ReadDir(t.Directory)
	// if err != nil {
	// 	return fmt.Errorf("failed to read project directory: %v", err)
	// }
	// t.Files = []string{}
	t.PRT = strings.TrimPrefix(getPRT(t.Directory), "_")
	// for _, file := range files {
	// 	if file.IsDir() {
	// 		continue
	// 	}
	// 	switch file.Name() {
	// 	default:
	// 		srcFile := joinPath(t.Directory, file.Name())
	// 		mp := ump.NewProfile()
	// 		switch err := mp.ConsumeFile(srcFile); err {
	// 		case nil:
	// 		default:
	// 			ok := false
	// 			if strings.HasPrefix(err.Error(), "file empty") {
	// 				ok = true
	// 			}
	// 			if !ok {
	// 				return fmt.Errorf("failed to scan source: %v", err)
	// 			}
	// 		}
	// 		t.MediaFiles[srcFile] = mediasource.NewSourceMedia(mp)

	// 	}
	// }
	source := taskMeta{}
	t.PRT = getPRT(t.Directory)
	if meta, ok := t.SignalFiles["metadata"]; ok {
		data, err := os.ReadFile(meta)
		if err != nil {
			return fmt.Errorf("failed to read: %v", err)
		}
		if err := json.Unmarshal(data, &source); err != nil {
			return fmt.Errorf("failed to unmarshal: %v", err)
		}
		project := prj.SearchByGUID(source.GUID)
		t.AmediaGUID = source.GUID
		t.AmediaTitleRus = project.RusTitle
		t.AmediaTitleOri = project.OriginalTitle
		t.Type = "MOV"
		if source.GUID != project.GUID {
			t.Type = "SER"
			t.Season, t.Episode = project.SeasonEpisode(t.AmediaGUID)
		}
		if source.File.Serid != "" {
			t.AmediaFileKey = source.File.Serid
		}
		t.OUTBASE = constructOutbase(t)

	} else {
		t.OUTBASE = filepath.Base(t.Directory)
		return fmt.Errorf("no metadata present")
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

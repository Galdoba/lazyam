package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Galdoba/appcontext/logmanager"
	"github.com/Galdoba/lazyam/internal/appmodule/config"
	"github.com/Galdoba/lazyam/internal/mediasource"
	"github.com/Galdoba/lazyam/internal/projectdata"
	"github.com/Galdoba/lazyam/pkg/notify"
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

func (t *Task) FillMetatada(prj *projectdata.Projects, logger *logmanager.Logger) error {

	list, err := getActualFileKeysCandidates(t.Directory)
	if err != nil {
		return fmt.Errorf("failed to get filekey candidates: %v", err)
	}
	if len(list) == 0 {
		logger.Warnf("no keys found: %v", t.Directory)
		return nil
	}
	ketSet := ""

	project := projectdata.AmediaProject{}
	fallbackMode := true
	for _, key := range list {
		project, err = prj.SearchMasterFileByAmediaFileKey(key)
		if err != nil {
			logger.Warnf("key %v: not found in global data", key)
			notify.SendErrorNoGlobalMetadataProvided(filepath.Base(t.Directory) + "_" + key)
			ketSet = key
		} else {
			logger.Infof("key %v: found in global data", key)
			t.AmediaFileKey = key
			t.AmediaGUID = project.GUID
			t.Season, t.Episode = project.SeasonEpisodeByFilekey(key)
			fallbackMode = false
			break
		}
		//////////
		project, err = SearchLocalByAmediaFileKey(t.SignalFiles["metadata"], key)
		if err != nil {
			logger.Warnf("key %v: search failed: %v", key, err.Error())
			// notify.SendErrorNoLocalMetadataProvided(filepath.Base(t.Directory) + "_" + key)
			ketSet = key
		} else {
			logger.Infof("key %v: found in local data", key)
			t.AmediaFileKey = key
			t.AmediaGUID = project.GUID
			t.AmediaTitleOri = project.OriginalTitle
			t.AmediaTitleRus = project.RusTitle
			t.Season, t.Episode = project.SeasonEpisodeByFilekey(key)
			fallbackMode = false
			break
		}

	}
	t.PRT = strings.TrimPrefix(getPRT(t.Directory), "_")
	t.PRT = getPRT(t.Directory)
	switch fallbackMode {
	case true:
		logger.Warnf("fallback %v to %s transcoding method", "no metadata", t.Directory)
		t.PRT = strings.TrimPrefix(getPRT(t.Directory), "_")
		t.PRT = getPRT(t.Directory)
		notify.SendErrorNoMetadata(filepath.Base(t.Directory) + "_" + ketSet)
	case false:
		if t.AmediaTitleOri == "" {
			t.AmediaTitleOri = project.OriginalTitle
		}
		if t.AmediaTitleRus == "" {
			t.AmediaTitleRus = project.RusTitle
		}
		keys := append(project.GUID_Candidates, t.AmediaFileKey)
		if t.Season < 1 {
			t.Season = int(prj.SearchSeason(keys...))
		}
		if t.Episode < 1 {
			t.Episode = int(prj.SearchEpisode(keys...))
		}
		if t.AmediaTitleOri == "" || t.AmediaTitleRus == "" {
			t.AmediaTitleRus, t.AmediaTitleOri = prj.SearchTitles(keys...)
		}

	}
	t.OUTBASE = constructOutbase(t)
	t.INBASE = constructInbase(t)
	return nil
}

func getActualFileKeysCandidates(dir string) ([]string, error) {
	list := []string{}
	fi, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}
fileLoop:
	for _, f := range fi {
		if f.IsDir() {
			continue
		}
		ext := filepath.Ext(f.Name())
		base := strings.TrimSuffix(f.Name(), ext)
		switch ext {
		case ".json", ".idet", "lock", "":
			continue
		default:
			for _, has := range list {
				if base == has {
					continue fileLoop
				}
			}
			list = append(list, base)
		}

	}
	return list, nil
}

// func (t *Task) fillFromIndividualMeta(prj *projectdata.Projects, logger *logmanager.Logger) (projectdata.AmediaProject, error) {
// 	source := taskMeta{}
// 	project := projectdata.AmediaProject{}
// 	if meta, ok := t.SignalFiles["metadata"]; ok {
// 		data, err := os.ReadFile(meta)
// 		if err != nil {
// 			return project, fmt.Errorf("failed to read: %v", err)
// 		}
// 		fmt.Println("found data:")
// 		fmt.Println(string(data))
// 		if err := json.Unmarshal(data, &source); err != nil {
// 			os.Rename(meta, meta+".bad")
// 			return project, fmt.Errorf("failed to unmarshal file: %v (%v)", meta, err)
// 		}
// 		fmt.Println(source)
// 		project = prj.SearchByGUID(source.GUID)
// 		if project.GUID == "" {
// 			logger.Warnf("no data by guid: %v", source.TitleRus)
// 		}
// 		// if project == projectdata.AmediaProject{} {
// 		// 	fmt.Println("search by guid not found")
// 		// }
// 		t.AmediaGUID = source.GUID
// 		t.AmediaTitleOri = source.TitleOri
// 		t.AmediaTitleRus = source.TitleRus
// 		if source.GUID != project.GUID {
// 			t.Type = "SER"
// 			t.Season, t.Episode = project.SeasonEpisode(t.AmediaGUID)
// 		}
// 		if source.Serid != "" {
// 			t.AmediaFileKey = source.Serid
// 		}
// 	} else {
// 		return project, fmt.Errorf("metadata file absent")
// 	}
// 	return project, nil
// }

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

func (t *Task) AssesInterlaceReport(cfg *config.Config) error {
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
			return err
		}
		t.InderlaceScanned = true
		t.ProgressiveRatio = float64(idetData.interlaceScore) / 1000.0
		threshold := int(1000 * cfg.Processing.InterlaceThreshold)
		if idetData.interlaceScore < threshold {
			t.InterlaceDetected = true
		}
		return nil
	}
	return fmt.Errorf("scan not started")
}

type idetScan struct {
	frameCount     map[int]int
	interlaceScore int
	err            error
}

var idetReg = `(.*Neither: *)(\d+)(.*Top: *)(\d+)( *Bottom: *)(\d+)(.*TFF: *)(\d+)(.*BFF: *)(\d+)(.*Progressive: *)(\d+)(.*Undetermined: *)(\d+)(.*TFF: *)(\d+)(.*BFF: *)(\d+)(.*Progressive: *)(\d+)(.*Undetermined: *)(\d+)$`

func parseIdet(path string) (idetScan, error) {
	is := idetScan{}
	is.frameCount = make(map[int]int)
	data, err := os.ReadFile(path)
	if err != nil {
		return is, err
	}
	if len(data) < 20 {
		return is, fmt.Errorf("scan not finished")
	}
	lines := strings.Split(string(data), "\n")
	text := strings.Join(lines, "")
	re := regexp.MustCompile(idetReg)
	subs := re.FindStringSubmatch(text)
	if len(subs) != 23 {
		return is, fmt.Errorf("failed to parse idet report: %v", path)
	}
	for _, s := range subs {
		if val, err := strconv.Atoi(s); err == nil {
			is.frameCount[len(is.frameCount)] = val
		}
	}
	total := is.frameCount[7] + is.frameCount[8] + is.frameCount[9] + is.frameCount[10]
	if total == 0 {
		return is, fmt.Errorf("scan data is inconclussive")
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
		for l, lang := range v.Languages {
			suf = append(suf, "AUDIO"+strings.ToUpper(lang)+v.Layout[l])
		}
	}
	return suf
}

func SearchLocalByAmediaFileKey(path, filekey string) (projectdata.AmediaProject, error) {

	project := projectdata.AmediaProject{}
	if path == "" {
		return project, fmt.Errorf("no local metadata provided")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return project, fmt.Errorf("failed to read file %v: %v", path, err)
	}
	source, err := extractMeta(data)
	if err != nil {
		return project, fmt.Errorf("failed to extract metadata: %v", err)
	}
	project.GUID = source.TITLE_GUID
	project.RusTitle = source.TitleRus
	project.OriginalTitle = source.TitleOri
	project.GUID_Candidates = append(project.GUID_Candidates, source.GUID, source.SEASON_GUID, source.TITLE_GUID)
	seasNum := extractNumber(source.SeasonName)
	project.Seasons = append(project.Seasons, projectdata.Season{
		Actors:      "",
		CmsID:       0,
		Directors:   "",
		OrderNumber: int64(seasNum),
		GUID:        source.SEASON_GUID,
		Episodes: []projectdata.Episode{
			{
				GUID:        source.GUID,
				OrderNumber: source.Episode_Num,
			},
		},
	})
	return project, nil
}

func extractNumber(s string) int {
	ns := ""
	for _, chr := range strings.Split(s, "") {
		switch chr {
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			ns += chr
		}
	}

	n, _ := strconv.Atoi(ns)
	return n
}

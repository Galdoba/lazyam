package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Galdoba/lazyam/internal/mediasource"
	"github.com/Galdoba/lazyam/internal/projectdata"
	"github.com/Galdoba/lazyam/pkg/translit"
)

type Task struct {
	IsReady           bool                              `json:"ready"`
	Directory         string                            `json:"directory"`
	Files             []string                          `json:"files"`
	PRT               string                            `json:"prt"`
	ProcessingStage   int                               `json:"status"`
	SignalFiles       map[string]string                 `json:"signal files"`
	MediaFiles        map[string]mediasource.SourceFile `json:"media files"`
	AmediaTitleRus    string                            `json:"title (rus)"`
	AmediaTitleOri    string                            `json:"title (original)"`
	AmediaGUID        string                            `json:"guid"`
	Type              string                            `json:"project type"`
	Season            int                               `json:"season num"`
	Episode           int                               `json:"episode num"`
	AmediaFileKey     string                            `json:"filekey"`
	IsSport           bool                              `json:"is sport"`
	InderlaceScanned  bool                              `json:"interlace scanned"`
	InterlaceDetected bool                              `json:"interlace detected"`
	ProgressiveRatio  float64                           `json:"Progressive confirmed"`
	INBASE            string                            `json:"projected source name prefix"`
	OUTBASE           string                            `json:"projected output name prefix"`
	undefined         bool
}

func New(directory string) *Task {
	t := Task{}
	t.Directory = directory
	t.SignalFiles = make(map[string]string)
	t.MediaFiles = make(map[string]mediasource.SourceFile)
	return &t
}

func (t *Task) AssertReady() error {
	if t.IsReady {
		return nil
	}
	files, err := os.ReadDir(t.Directory)
	if err != nil {
		return fmt.Errorf("failed to read project directory: %v", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		srcFile := joinPath(t.Directory, file.Name())
		fl, err := os.OpenFile(srcFile, os.O_WRONLY|os.O_EXCL, 0644)
		if err != nil {
			t.IsReady = false
		}
		defer fl.Close()
	}
	t.IsReady = true
	return nil
}

func (t *Task) CollectSignals() error {
	files, err := os.ReadDir(t.Directory)
	if err != nil {
		return fmt.Errorf("failed to read project directory: %v", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		switch file.Name() {
		case "metadata.json":
			t.SignalFiles["metadata"] = joinPath(t.Directory, file.Name())
		case "lock":
			t.SignalFiles["lock"] = joinPath(t.Directory, file.Name())

		default:
			continue
		}
	}
	return nil
}

type taskMeta struct {
	GUID     string           `json:"guid,omitempty"`
	File     projectdata.File `json:"file,omitempty"`
	TitleOri string           `json:"original_title",omitempty"`
	TitleRus string           `json:"rus_title",omitempty"`

	/*
			"guid": "FRDAE0000",
		    "start_date": "2025-09-01T00:00:00+03:00",
		    "end_date": null,
		    "year": 2016,
		    "file": {
		        "serid": "MOV_00223_18",
		        "duration": 7259.0
		    },
		    "original_title": "Rules Don't Apply",
		    "rus_title": "Вне правил",
		    "age_restriction": null,
		    "genre": null,
		    "country": null,
		    "directors": null,
		    "actors": null,
		    "rus_description": "",
		    "imdb_id": null,
		    "kinopoisk_id": null,
		    "quote": null,
		    "quote_author": null
	*/
}

func getPRT(str string) string {
	str = strings.ToUpper(str)
	re := regexp.MustCompile(`(PRT[0-9]+)`)
	prt := re.FindString(str)
	if prt == "" {
		prt = fmt.Sprintf("PRT000000000000")
	}
	return prt
}

func constructOutbase(t *Task) string {
	title := t.AmediaTitleRus
	if title == "" {
		title = t.AmediaTitleOri
	}
	if title == "" {
		t.undefined = true
		return filepath.Base(t.Directory)
	}

	tags := []string{
		toTitle(translit.String(title, translit.RegisterLow())),
	}
	tags = appendNonEmpty(tags, seasEpisOutString(t.Season, t.Episode))
	tags = appendNonEmpty(tags, t.PRT)
	return strings.Join(tags, "_")
}

func constructInbase(t *Task) string {
	title := t.AmediaTitleRus
	if title == "" {
		title = t.AmediaTitleOri
	}
	if title == "" {
		return filepath.Base(t.Directory)
	}

	tags := []string{
		toTitle(translit.String(title, translit.RegisterLow())),
	}
	tags = appendNonEmpty(tags, seasEpisInString(t.Season, t.Episode))
	tags = appendNonEmpty(tags, t.PRT)
	return strings.Join(tags, "_")
}

func appendNonEmpty(sl []string, s string) []string {
	if s == "" {
		return sl
	}
	return append(sl, s)
}

func toTitle(s string) string {
	out := ""
	for i, letter := range strings.Split(s, "") {
		if i == 0 {
			letter = strings.ToUpper(letter)
		}
		out += letter

	}
	return out
}

func seasEpisInString(s, e int) string {
	out := ""
	if s > 0 {
		out += "s" + numToStr(s)
	}
	if e > 0 {
		out += "e" + numToStr(e)
	}
	return out

}
func seasEpisOutString(s, e int) string {
	out := ""
	if s > 0 {
		out += "s" + numToStr(s)
	}
	if e > 0 {
		out += "_" + numToStr(e)
	}
	return out

}

func numToStr(n int) string {
	s := fmt.Sprintf("%v", n)
	for len(s) < 2 {
		s = "0" + s
	}
	return s
}

func joinPath(str ...string) string {
	return filepath.ToSlash(filepath.Join(str...))
}

package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Galdoba/lazyam/internal/mediasource"
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
	GUID        string  `json:"guid,omitempty"`
	TITLE_GUID  string  `json:"title_guid,omitempty"`
	SEASON_GUID string  `json:"season_guid,omitempty"`
	TitleOri    string  `json:"original_title",omitempty"`
	TitleRus    string  `json:"rus_title",omitempty"`
	Serid       string  `json:"serid,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	SeasonName  string  `json:"orig_name,omitempty"`
	Season_Num  int64   `json:"season_num,omitempty"`
	Episode_Num int64   `json:"episode_num,omitempty"`
}

// ///////////////////////////////////
// AI GENERATED
// type taskMeta struct {
// 	GUID     string  `json:"guid,omitempty"`
// 	TitleOri string  `json:"original_title,omitempty"`
// 	TitleRus string  `json:"rus_title,omitempty"`
// 	Serid    string  `json:"serid,omitempty"`
// 	Duration float64 `json:"duration,omitempty"`
// }

// extractMeta извлекает метаданные из JSON с дополнительными проверками
func extractMeta(data []byte) (taskMeta, error) {
	var result taskMeta
	var unmarshaled interface{}

	// Проверка 1: Декодирование JSON
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		return result, fmt.Errorf("failed json unmarshaling: %v", err)
	}

	// Рекурсивно обходим структуру
	extractFields(unmarshaled, &result)

	// Проверка 2: Хотя бы одно поле должно быть заполнено
	if isEmptyMeta(result) {
		return result, errors.New("no usefull metadata found")
	}

	return result, nil
}

// isEmptyMeta проверяет, все ли поля структуры пустые
func isEmptyMeta(meta taskMeta) bool {
	return meta.GUID == "" &&
		meta.TitleOri == "" &&
		meta.TitleRus == "" &&
		meta.Serid == "" &&
		meta.Duration == 0 &&
		meta.SeasonName == "" &&
		meta.Season_Num == 0 &&
		meta.Episode_Num == 0
}

// Остальные функции остаются без изменений
func extractFields(data interface{}, result *taskMeta) {
	if data == nil {
		return
	}

	value := reflect.ValueOf(data)
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()

			// Проверяем и разыменовываем интерфейсы
			if k.IsValid() && k.Kind() == reflect.Interface {
				k = k.Elem()
			}
			if v.IsValid() && v.Kind() == reflect.Interface {
				v = v.Elem()
			}

			if k.IsValid() && k.Kind() == reflect.String {
				key := k.String()
				if v.IsValid() {
					processKeyValue(key, v.Interface(), result)
				}
			}

			// Рекурсивно обходим вложенные структуры
			if v.IsValid() {
				extractFields(v.Interface(), result)
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			if item.IsValid() && item.Kind() == reflect.Interface {
				item = item.Elem()
			}
			if item.IsValid() {
				extractFields(item.Interface(), result)
			}
		}

	default:
		// Для простых типов ничего не делаем
	}
}

func processKeyValue(key string, value interface{}, result *taskMeta) {
	if value == nil {
		return
	}

	switch key {
	case "guid":
		if str, ok := value.(string); ok {
			result.GUID = str
		}
	case "title_guid":
		if str, ok := value.(string); ok {
			result.TITLE_GUID = str
		}
	case "season_guid":
		if str, ok := value.(string); ok {
			result.SEASON_GUID = str
		}
	case "original_title":
		if str, ok := value.(string); ok {
			result.TitleOri = str
		}
	case "rus_title":
		if str, ok := value.(string); ok {
			result.TitleRus = str
		}
	case "serid":
		if str, ok := value.(string); ok {
			result.Serid = str
		}
	case "duration":
		switch v := value.(type) {
		case float64:
			result.Duration = v
		case int:
			result.Duration = float64(v)
		case int64:
			result.Duration = float64(v)
		case float32:
			result.Duration = float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				result.Duration = f
			}
		}
	case "orig_name":
		if str, ok := value.(string); ok {
			result.SeasonName = str
		}
	case "season_num":
		switch v := value.(type) {
		case int64:
			result.Season_Num = v
		case float64:
			result.Season_Num = int64(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				result.Season_Num = int64(i)
			}
		}
	case "episode_num", "order_number":
		switch v := value.(type) {
		case int64:
			result.Episode_Num = v
		case float64:
			result.Episode_Num = int64(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				result.Episode_Num = int64(i)
			}
		}
	}
}

////////////////////////////////////

func getPRT(str string) string {
	str = strings.ToUpper(str)
	re := regexp.MustCompile(`(PRT[0-9]+)`)
	prt := re.FindString(str)
	if prt == "" {
		prt = fmt.Sprintf("PRT%v000000", today())
	}
	for len(prt) < 15 {
		prt += "0"
	}
	return prt
}

func today() string {
	return time.Now().Format("060102")
}

func constructOutbase(t *Task) string {
	title := t.AmediaTitleRus
	if title == "" {
		title = t.AmediaTitleOri
	}
	if title == "" {
		//t.undefined = true
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

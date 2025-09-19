package projectdata

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Galdoba/appcontext/logmanager"
	"github.com/Galdoba/lazyam/internal/analitycs"
	"github.com/Galdoba/lazyam/internal/appmodule/config"
)

// Projects - Represents list of project metadata.
type Projects struct {
	Pool       map[string]AmediaProject `json:"project list"`
	LastUpdate time.Time                `json:"last updated"`
}

// AmediaProject - Represents lazyam unified project metadata format.
type File struct {
	Serid    string  `json:"serid,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

type Season struct {
	Actors            string    `json:"actors,omitempty"`
	CmsID             int64     `json:"cms_id,omitempty"`
	Directors         string    `json:"directors,omitempty"`
	EndDate           string    `json:"end_date,omitempty"`
	Episodes          []Episode `json:"episodes,omitempty"`
	GUID              string    `json:"guid,omitempty"`
	OrderNumber       int64     `json:"order_number,omitempty"`
	OrigName          string    `json:"orig_name,omitempty"`
	RusName           string    `json:"rus_name,omitempty"`
	SeasonDescription string    `json:"season_description,omitempty"`
	StartDate         string    `json:"start_date,omitempty"`
	YearsI            int64     `json:"years (int),omitempty"`
	YearsS            string    `json:"years (string),omitempty"`
}

type Episode struct {
	CmsID               int64  `json:"cms_id,omitempty"`
	EndDate             string `json:"end_date,omitempty"`
	EpisodeSynopsis     string `json:"episode_synopsis,omitempty"`
	File                File   `json:"file,omitempty"`
	GUID                string `json:"guid,omitempty"`
	OrderNumber         int64  `json:"order_number,omitempty"`
	OriginalEpisodeName string `json:"original_episode_name,omitempty"`
	RusEpisodeName      string `json:"rus_episode_name,omitempty"`
	StartDate           string `json:"start_date,omitempty"`
	Year                int64  `json:"year,omitempty"`
}

type AmediaProject struct {
	Type                string   `json:"project_type,omitempty"`
	Actors              string   `json:"actors,omitempty"`
	AgeRestriction      string   `json:"age_restriction,omitempty"`
	CmsID               int64    `json:"cms_id,omitempty"`
	Country             string   `json:"country,omitempty"`
	Directors           string   `json:"directors,omitempty"`
	EndDate             string   `json:"end_date,omitempty"`
	Seasons             []Season `json:"seasons,omitempty"`
	File                File     `json:"file,omitempty"`
	Genre               string   `json:"genre,omitempty"`
	GUID                string   `json:"guid,omitempty"`
	ImdbID              string   `json:"imdb_id,omitempty"`
	KinopoiskID         string   `json:"kinopoisk_id,omitempty"`
	OriginalTitle       string   `json:"original_title,omitempty"`
	Quote               string   `json:"quote,omitempty"`
	QuoteAuthor         string   `json:"quote_author,omitempty"`
	RusDescription      string   `json:"rus_description,omitempty"`
	RusTitle            string   `json:"rus_title,omitempty"`
	StartDate           string   `json:"start_date,omitempty"`
	Year                int64    `json:"year,omitempty"`
	OriginalBroadcaster string   `json:"original_broadcaster,omitempty"`
	GUID_Candidates     []string `json:"guidcandidates,omitempty"`
}

func NewProjects() *Projects {
	p := Projects{}
	p.Pool = make(map[string]AmediaProject)
	return &p
}

func (original *Projects) Update(cfg *config.Config, log *logmanager.Logger) error {
	if original == nil {
		original = &Projects{}
	}
	paths := cfg.Declarations.MetadataFiles
	for _, sourceFile := range paths {
		metadataUpdateTime, err := fileModTime(sourceFile)
		if err != nil {
			log.Warnf("failed to asses file modification time: %v", err)
			log.Noticef("update rejected")
		}
		switch wantUpdate(original, metadataUpdateTime) {
		case false:
			log.Tracef("skip update")
			continue
		case true:
			log.Tracef("want update")
		}
		// want, err := wantUpdate(cfg.Declarations.ProjectCacheFile, sourceFile)
		// if err != nil {
		// }
		// if !want && len(original.Pool) > 0 {
		// 	continue
		// }
		updated, err := updateProjectData(sourceFile, cfg.Declarations.ProjectCacheFile, log)
		if err != nil {
			log.Warnf("failed to update project data: %v", err)
			continue
		}
		if updated == nil {
			continue
		}
		for _, new := range updated.Pool {
			original.inject(new, log)
		}
		log.Infof("updated: %v from %v", cfg.Declarations.ProjectCacheFile, sourceFile)
		if err := original.Save(cfg.Declarations.ProjectCacheFile); err != nil {
			log.Errorf("cache saving failed: %v", nil)
		}
		original.LastUpdate = metadataUpdateTime
	}
	return nil
}

func (ap AmediaProject) Name() string {
	if ap.RusTitle != "" {
		return fmt.Sprintf("%v [%v]", ap.RusTitle, ap.GUID)
	}
	if ap.OriginalTitle != "" {
		return fmt.Sprintf("%v [%v]", ap.OriginalTitle, ap.GUID)
	}
	return ap.GUID
}

func wantUpdate(projects *Projects, metadataUpdateTime time.Time) bool {
	if len(projects.Pool) == 0 {
		return true
	}
	return metadataUpdateTime.After(projects.LastUpdate)
}

func fileModTime(path string) (time.Time, error) {
	f, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return f.ModTime(), nil
}

func (project *Projects) inject(new AmediaProject, log *logmanager.Logger) {
	for _, old := range project.Pool {
		key := old.Name()
		if old.Name() != new.Name() {
			continue
		}
		equal, err := equalProjectData(old, new)
		if err != nil {
			log.Errorf("project comparison failed: %v", err)
			log.Infof("skip update: %v", old.Name())
			continue
		}
		switch equal {
		case false:
			project.Pool[key] = new
			log.Infof("%v: metadata updated", old.Name())
		case true:
		}
		return
	}
	project.Pool[new.Name()] = new
}

func equalProjectData(old, new AmediaProject) (bool, error) {
	dataOld, err := json.Marshal(&old)
	if err != nil {
		return false, err
	}
	dataNew, err := json.Marshal(&new)
	if err != nil {
		return false, err
	}
	return string(dataOld) == string(dataNew), nil
}

func updateProjectData(source, destination string, log *logmanager.Logger) (*Projects, error) {
	log.Infof("update started: %v", destination)
	projects := AmediaProjectMetadata{}
	converted := Projects{}
	converted.Pool = make(map[string]AmediaProject)
	bt, err := os.ReadFile(source)
	if err != nil {
		log.Errorf("failed to read file: %v", err)
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	if err := json.Unmarshal(bt, &projects); err != nil {
		log.Errorf("failed to unmarshal file: %v", err)
		return nil, fmt.Errorf("failed to unmarshal file: %v", err)
	}
	for _, film := range projects.Movies {
		pr := AmediaProject{}
		bt, err := json.MarshalIndent(film, "", "  ")
		if err != nil {
			log.Warnf("failed to marshal provider data: cmsID:%v (%v)", film.CmsID, film.RusTitle)
			continue
		}
		if err := json.Unmarshal(bt, &pr); err != nil {
			log.Warnf("failed to unmarshal data: cmsID:%v (%v)", film.CmsID, film.RusTitle)
		}
		converted.Pool[pr.Name()] = pr
	}
	for _, ser := range projects.Series {
		pr := AmediaProject{}
		bt, err := json.MarshalIndent(ser, "", "  ")
		if err != nil {
			log.Warnf("failed to marshal provider data: cmsID:%v (%v) : %v", ser.CmsID, ser.RusTitle, err)
			continue
		}
		if err := json.Unmarshal(bt, &pr); err != nil {
			log.Errorf("failed to unmarshal data: %v", err.Error())
		}
		converted.Pool[pr.Name()] = pr
	}
	data, err := json.MarshalIndent(&converted, "", "  ")
	if err != nil {
		log.Errorf("failed to marshal converted data: %v positions", len(converted.Pool))
		return nil, fmt.Errorf("failed to marshal converted data: %v positions", len(converted.Pool))
	}
	if err := os.WriteFile(destination, data, 0644); err != nil {
		log.Errorf("failed to write converted data to local cache: %v", err)
		return nil, fmt.Errorf("failed to write converted data to local cache: %v", err)
	}
	// log.Debugf("update completed")
	analitycs.UpdateCompleted(log)

	return &converted, nil
}

func (list *Projects) Save(path string) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project data")
	}
	return os.WriteFile(path, data, 0644)
}

func (pr *Projects) SearchByGUID(guid string) AmediaProject {
	notFound := AmediaProject{}
	for _, prj := range pr.Pool {
		if prj.GUID == guid {
			return prj
		}
		for _, season := range prj.Seasons {
			if season.GUID == guid {
				return prj
			}
			for _, episode := range season.Episodes {
				if episode.GUID == guid {
					return prj
				}
			}
		}
	}
	return notFound
}

func (prj AmediaProject) SeasonEpisodeByGUID() (int, int) {
	for _, guid := range prj.GUID_Candidates {
		// if len(prj.Seasons) == 1 && len(prj.Seasons[0].Episodes) == 1 {
		// 	fmt.Println("one escape", int(prj.Seasons[0].OrderNumber), int(prj.Seasons[0].Episodes[0].OrderNumber))
		// 	return int(prj.Seasons[0].OrderNumber), int(prj.Seasons[0].Episodes[0].OrderNumber)
		// }
		// if prj.GUID == guid {
		// 	return 0, 0
		// }
		for _, season := range prj.Seasons {
			if season.GUID == guid {
				return int(season.OrderNumber), 0
			}
			for _, episode := range season.Episodes {
				if episode.GUID == guid {

					return int(season.OrderNumber), int(episode.OrderNumber)
				}
			}
		}
	}
	return -1, -1
}

func (prj AmediaProject) SeasonEpisodeByFilekey(filekey string) (int, int) {
	if len(prj.Seasons) == 1 && len(prj.Seasons[0].Episodes) == 1 {
		return int(prj.Seasons[0].OrderNumber), int(prj.Seasons[0].Episodes[0].OrderNumber)
	}
	if prj.GUID == filekey {
		return 0, 0
	}
	for _, season := range prj.Seasons {
		for _, episode := range season.Episodes {
			if episode.File.Serid == filekey {
				return int(season.OrderNumber), int(episode.OrderNumber)
			}
		}
	}
	return -1, -1
}

func (prj *Projects) SearchMasterFileByAmediaFileKey(filekey string) (AmediaProject, error) {
	notFound := AmediaProject{}
	for _, pr := range prj.Pool {
		if pr.File.Serid == filekey {
			return pr, nil
		}
		for _, season := range pr.Seasons {
			for _, episode := range season.Episodes {
				if episode.File.Serid == filekey {
					return pr, nil
				}
			}
		}
	}
	return notFound, fmt.Errorf("not found")
}

func (prj *Projects) SearchSeason(keys ...string) int64 {
	found := -1
	for _, key := range keys {
		for _, pr := range prj.Pool {
			if pr.GUID == key {
				return 0
			}
			if pr.File.Serid == key {
				return 0
			}
			for _, season := range pr.Seasons {
				if season.GUID == key {
					return season.OrderNumber
				}
				for _, episode := range season.Episodes {
					if episode.GUID == key {
						return season.OrderNumber
					}
					if episode.File.Serid == key {
						return season.OrderNumber
					}
				}
			}
		}
	}
	return int64(found)
}

func (prj *Projects) SearchEpisode(keys ...string) int64 {
	found := -1
	for _, key := range keys {
		for _, pr := range prj.Pool {
			if pr.GUID == key {
				return 0
			}
			if pr.File.Serid == key {
				return 0
			}
			for _, season := range pr.Seasons {

				if season.GUID == key {
					return 0
				}
				for _, episode := range season.Episodes {
					if episode.GUID == key {
						return episode.OrderNumber
					}
					if episode.File.Serid == key {
						return episode.OrderNumber
					}
				}
			}
		}
	}
	return int64(found)
}

func (prj *Projects) SearchTitles(keys ...string) (string, string) {
	for _, key := range keys {
		for _, pr := range prj.Pool {
			if pr.GUID == key {
				return pr.RusTitle, pr.OriginalTitle
			}
			if pr.File.Serid == key {
				return pr.RusTitle, pr.OriginalTitle
			}
			for _, season := range pr.Seasons {
				if season.GUID == key {
					return pr.RusTitle, pr.OriginalTitle
				}
				for _, episode := range season.Episodes {
					if episode.GUID == key {
						return pr.RusTitle, pr.OriginalTitle
					}
					if episode.File.Serid == key {
						return pr.RusTitle, pr.OriginalTitle
					}
				}
			}
		}
	}
	return "", ""
}

package analitycs

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Galdoba/golog"
	"github.com/Galdoba/lazyam/internal/config"
)

type ProcessStats struct {
	enabled             bool
	path                string
	ProjectsUpdated     time.Time `json:"metadata last updated at"`
	ProjectFilesUpdated int       `json:"times metadata updated"`
}

var NoStats = ProcessStats{
	enabled: false,
}

var analitycTracker = &ProcessStats{}

func StartTracker(cfg *config.Config, logger *golog.Logger) {
	if !cfg.Analitycs.Enabled {
		logger.Noticef("analitycs disabled")
		analitycTracker = &NoStats
		return
	}
	stats := ProcessStats{}
	path := cfg.Analitycs.StatisticFile
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Errorf("failed to read stats file %v: %v", path, err)
		logger.Warnf("fall back to %v mode", "no stats")
		analitycTracker = &NoStats
		return
	}
	if err := json.Unmarshal(data, &stats); err != nil {
		logger.Errorf("failed to unmarshal stats data: %v", err)
		logger.Warnf("fall back to %v mode", "no stats")
		analitycTracker = &NoStats
		return
	}
	stats.path = path
	analitycTracker = &stats
}

func UpdateCompleted(log *golog.Logger) {
	if !analitycTracker.enabled {
		return
	}
	analitycTracker.ProjectsUpdated = time.Now()
	analitycTracker.ProjectFilesUpdated++
	data, err := json.MarshalIndent(analitycTracker, "", "  ")
	if err != nil {
		log.Errorf("statistics marshalling failed: %v", err)
		return
	}
	if err := os.WriteFile(analitycTracker.path, data, 0644); err != nil {
		log.Errorf("statistics saving failed: %v", err)
		return
	}
	log.Infof("cache update completed")
}

func LastUpdateTime() time.Time {
	return analitycTracker.ProjectsUpdated
}

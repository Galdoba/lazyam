package task

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Galdoba/appcontext/logmanager"
	"github.com/Galdoba/lazyam/internal/appmodule/config"
)

type TaskList struct {
	Tasks map[string]*Task `json:"tasks"`
}

func NewTaskList() *TaskList {
	tl := TaskList{}
	tl.Tasks = make(map[string]*Task)
	return &tl
}

func (tl *TaskList) Update(cfg *config.Config, log *logmanager.Logger) error {
	path := cfg.Declarations.TaskCacheFile
	data, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("failed to read cache file: %v", path)
		return fmt.Errorf("failed to read cache file: %v", path)
	}
	if err := json.Unmarshal(data, tl); err != nil {
		log.Errorf("failed to unmarshal task cache: %v", err.Error())
		return fmt.Errorf("failed to read cache file: %v", err.Error())
	}

	return nil
}

func (tl *TaskList) Save(path string) error {
	data, err := json.MarshalIndent(tl, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project data")
	}
	return os.WriteFile(path, data, 0644)
}

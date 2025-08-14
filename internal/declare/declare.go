package declare

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	APP_NAME        = "lazyam"
	LOG_FILE        = "lazyam.log"
	STATISTICS_FILE = "statistics.json"
	PROJECTS_FILE   = "project_data.json"
	TASKS_FILE      = "tasks_data.json"
)

func DefaultCacheDirWithFile(file string) string {
	return filepath.ToSlash(filepath.Join(home(), ".cache", APP_NAME, file))
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	return h
}

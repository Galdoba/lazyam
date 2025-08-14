package actionstage

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/Galdoba/lazyam/internal/projectdata"
	"github.com/Galdoba/lazyam/internal/task"
	lazyerror "github.com/Galdoba/lazyam/pkg/error"
)

// LoadMetadataCache - Load project metadata from cache file and return error.
func LoadMetadataCache(path string) (*projectdata.Projects, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, lazyerror.NewExpected(err, "cache file %v not exist", path)
		}
		return nil, lazyerror.NewUnexpected("cache file reading failed: %v", err)
	}
	prj := projectdata.Projects{}
	if err := json.Unmarshal(data, &prj); err != nil {
		return nil, lazyerror.NewUnexpected("failed to unmarshal metadata: %v", err.Error())
	}
	return &prj, nil
}

// LoadProjectsCache - Load tasks data from cache file and return error.
func LoadProjectsCache(path string) (*task.TaskList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, lazyerror.NewExpected(err, "cache file %v not exist", path)
		}
		return nil, lazyerror.NewUnexpected("cache file reading failed: %v", err)
	}
	tasks := task.TaskList{}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, lazyerror.NewUnexpected("failed to unmarshal projects data: %v", err.Error())
	}
	return &tasks, nil

}

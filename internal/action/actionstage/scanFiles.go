package actionstage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Galdoba/golog"
	"github.com/Galdoba/lazyam/internal/config"
	"github.com/Galdoba/lazyam/internal/projectdata"
	"github.com/Galdoba/lazyam/internal/task"
)

func ScanSources(cfg *config.Config, log *golog.Logger, prj *projectdata.Projects) ([]*task.Task, error) {
	tasks := []*task.Task{}
	fi, err := os.ReadDir(cfg.Declarations.InputDirectory)
	if err != nil {
		log.Errorf("failed to read input directory: %v", err)
		return nil, fmt.Errorf("failed to read input directory: %v", err)
	}
	for _, f := range fi {
		if !f.IsDir() {
			continue
		}
		tsk := &task.Task{
			Directory: filepath.Join(cfg.Declarations.InputDirectory, f.Name()),
		}
		switch err := tsk.FillMetatada(prj); err {
		case nil:
			log.Debugf("metadata filled for %v", tsk.OUTBASE)
		default:
			log.Warnf("failed to fill %v project metadata: %v", tsk.OUTBASE, err.Error())
		}
		tasks = append(tasks, tsk)

	}
	return tasks, nil
}

func ListActiveTasks(cfg *config.Config) ([]string, error) {
	tasks := []string{}
	fi, err := os.ReadDir(cfg.Declarations.InputDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to read input directory: %v", err)
	}
	for _, f := range fi {
		if !f.IsDir() {
			continue
		}
		tasks = append(tasks, filepath.ToSlash(filepath.Join(cfg.Declarations.InputDirectory, f.Name())))
	}
	return tasks, nil
}

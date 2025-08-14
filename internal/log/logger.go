package log

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Galdoba/golog"
	"github.com/Galdoba/lazyam/internal/config"
	"github.com/Galdoba/lazyam/internal/declare"
)

type Logger struct {
	logger       *golog.Logger
	enabled      bool
	logDirectory string
	filePrefix   string
	rotation     string
}

// Start - Create new logger based on config options.
func Start(logging config.Logging) (*golog.Logger, error) {
	l := Logger{}
	l.enabled = logging.Enabled
	if !logging.Enabled {
		return golog.New(), nil
	}
	l = Logger{
		enabled: logging.Enabled,
		//TODO: rotation: logging.FileRotation,
	}
	logFile := logging.FilePath
	if logFile == "" {
		logFile = declare.DefaultCacheDirWithFile(declare.LOG_FILE)
	}
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to enshure log directory: %v", err)
	}

	consoleLevel := golog.LevelTrace
	fileLevel := golog.LevelTrace
	logName := filepath.Base(logFile)
	lvls, strLvls := levels()
	for i, lvl := range lvls {
		if logging.ConsoleLevel == strLvls[i] {
			consoleLevel = lvl
		}
		if logging.FileLevel == strLvls[i] {
			fileLevel = lvl
		}
	}

	consoleHandler := golog.NewHandler(os.Stderr, consoleLevel, golog.NewTextFormatter(golog.WithColor(logging.ConsoleColor), golog.WithTimePrecision(3), golog.WithLevelTag(true)))
	fmt.Println("set path", filepath.Join(logDir, logName))
	file, err := os.OpenFile(filepath.Join(logDir, logName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	// defer file.Close()
	fileHandler := golog.NewHandler(file, fileLevel, golog.NewTextFormatter(golog.WithTimePrecision(3), golog.WithLevelTag(true)))
	l.logger = golog.New(golog.WithLevel(golog.LevelTrace))
	l.logger.AddHandler(consoleHandler)
	l.logger.AddHandler(fileHandler)

	return l.logger, nil
}

func levels() ([]golog.LogLevel, []string) {
	return golog.ListLevels()
}

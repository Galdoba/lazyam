package log

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Galdoba/appcontext/logmanager"
	"github.com/Galdoba/lazyam/internal/appmodule/config"
	"github.com/Galdoba/lazyam/internal/declare"
)

type Logger struct {
	logger       *logmanager.Logger
	enabled      bool
	logDirectory string
	filePrefix   string
	rotation     string
}

// Start - Create new logger based on config options.
func Start(logging config.Logging) (*logmanager.Logger, error) {
	l := Logger{}
	l.enabled = logging.Enabled
	if !logging.Enabled {
		return logmanager.New(), nil
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

	consoleLevel := logmanager.LevelTrace
	fileLevel := logmanager.LevelTrace
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

	consoleHandler := logmanager.NewHandler(logmanager.Stderr, consoleLevel, logmanager.NewTextFormatter(logmanager.WithColor(logging.ConsoleColor), logmanager.WithTimePrecision(3), logmanager.WithLevelTag(true)))
	fmt.Println("set path", filepath.Join(logDir, logName))
	fileHandler := logmanager.NewHandler(logFile, fileLevel, logmanager.NewTextFormatter(logmanager.WithTimePrecision(3), logmanager.WithLevelTag(true)))
	l.logger = logmanager.New(logmanager.WithLevel(logmanager.LevelTrace))
	l.logger.AddHandler(consoleHandler)
	l.logger.AddHandler(fileHandler)

	return l.logger, nil
}

func levels() ([]logmanager.LogLevel, []string) {
	return logmanager.ListLevels()
}

package config

import (
	"path/filepath"

	"github.com/Galdoba/appcontext/xdg"
	"github.com/Galdoba/lazyam/internal/declare"
)

type Config struct {
	Declarations Declarations `toml:"declarations"`
	Processing   Processing   `toml:"processing"`
	Analitycs    Analitycs    `toml:"analitycs"`
	Logging      Logging      `toml:"logging"`
}

type Declarations struct {
	InputDirectory        string   `toml:"input root directory"`
	OutputDirectory       string   `toml:"output directory"`
	WorkerReportDirectory string   `toml:"worker feedback directory"`
	MetadataFiles         []string `toml:"metadata files"`
	ProjectCacheFile      string   `toml:"projects data file"`
	TaskCacheFile         string   `toml:"tasks data file"`
}

type Processing struct {
	ConfigReload          bool `toml:"reload config on each cycle"`
	DormantMode           int  `toml:"sleep time between cycles (seconds)"`
	CycleLockAutoremove   int  `toml:"autoremove cycle lock after (seconds)"`
	ProjectLockAutoremove int  `toml:"autoremove project lock after (seconds)"`
}

type Analitycs struct {
	Enabled          bool            `toml:"enabled"`
	TrackStatisctics map[string]bool `toml:"statistic tracking"`
	StatisticFile    string          `toml:"statistics file"`
}

type Logging struct {
	Enabled      bool   `toml:"enabled"`
	FilePath     string `toml:"log file path"`
	FileRotation string `toml:"log file rotation [none/dayly/weekly/monthly]"`
	ConsoleColor bool   `toml:"console color output"`
	ConsoleLevel string `toml:"write messages above level (console)"`
	FileLevel    string `toml:"write messages above level (file)"`
}

func Default(paths *xdg.ProgramPaths) Config {
	return Config{
		Declarations: Declarations{
			InputDirectory:        `//192.168.31.4/buffer/IN/@AMEDIA_IN/`,
			OutputDirectory:       "//192.168.31.4/buffer/IN/",
			WorkerReportDirectory: `//192.168.31.4/buffer/IN/@AMEDIA_IN/__reports/`,
			MetadataFiles:         []string{`//192.168.31.4/buffer/IN/@AMEDIA_IN/metadata.json`},
			// ProjectCacheFile:      declare.DefaultCacheDirWithFile(declare.PROJECTS_FILE),
			ProjectCacheFile: filepath.Join(paths.CacheDir(), declare.PROJECTS_FILE),
			// TaskCacheFile:    declare.DefaultCacheDirWithFile(declare.TASKS_FILE),
			TaskCacheFile: filepath.Join(paths.CacheDir(), declare.TASKS_FILE),
		},
		Processing: Processing{
			ConfigReload:          false,
			DormantMode:           3,
			CycleLockAutoremove:   9,
			ProjectLockAutoremove: 27,
		},
		Analitycs: Analitycs{
			Enabled: false,
			TrackStatisctics: map[string]bool{
				"lock detected (cycle)":   true,
				"lock detected (project)": true,
				"lock removed (cycle)":    true,
				"lock removed (project)":  true,
			},
			// StatisticFile: declare.DefaultCacheDirWithFile(declare.STATISTICS_FILE),
			StatisticFile: filepath.Join(paths.PersistentDataDir(), declare.STATISTICS_FILE),
		},
		Logging: Logging{
			Enabled: true,
			// FilePath:     declare.DefaultCacheDirWithFile(declare.LOG_FILE),
			FilePath:     paths.LogFile(),
			FileRotation: "none",
			ConsoleColor: true,
			ConsoleLevel: "trace",
			FileLevel:    "trace",
		},
	}
}

// Load - Load actual config file or create default on first run.
// func Load() (*Config, error) {
// 	cfg := Default()
// 	cm, err := configmanager.New(declare.APP_NAME, cfg)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create config manager: %v", err)
// 	}
// 	if err := cm.Load(); err != nil {
// 		return nil, fmt.Errorf("failed to load config: %v", err)
// 	}
// 	return &cfg, nil
// }

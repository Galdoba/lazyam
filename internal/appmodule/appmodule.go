package appmodule

import (
	"fmt"

	"github.com/Galdoba/appcontext/configmanager"
	"github.com/Galdoba/appcontext/logmanager"
	"github.com/Galdoba/appcontext/xdg"
	"github.com/Galdoba/lazyam/internal/appmodule/config"
	"github.com/Galdoba/lazyam/internal/appmodule/log"
)

type AppContext struct {
	Paths  *xdg.ProgramPaths
	Config *config.Config
	Log    *logmanager.Logger
}

func Initiate(appName string) (*AppContext, error) {
	ac := AppContext{}
	ac.Paths = xdg.New(appName)
	cm, err := configmanager.New(appName, config.Default(ac.Paths))
	if err != nil {
		return nil, fmt.Errorf("config: %v", err)
	}
	if err := cm.Load(); err != nil {
		return nil, fmt.Errorf("config: %v", err)
	}
	ac.Config = cm.Config()
	logger, err := log.Start(ac.Config.Logging)
	if err != nil {
		return nil, fmt.Errorf("logger: %v", err)
	}
	ac.Log = logger
	return &ac, nil
}

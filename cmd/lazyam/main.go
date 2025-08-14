package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/Galdoba/lazyam/internal/action"
	"github.com/Galdoba/lazyam/internal/config"
	"github.com/Galdoba/lazyam/internal/declare"
	"github.com/Galdoba/lazyam/internal/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v", err)
		os.Exit(1)
	}
	logger, err := log.Start(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start logger: %v", err)
		os.Exit(1)
	}
	cmd := cli.Command{
		Name:        declare.APP_NAME,
		Version:     "0.0.0",
		Description: "Automatic amedia content transcoding.",
		Action:      action.Run,

		Metadata: map[string]interface{}{
			"config": cfg,
			"logger": logger,
		},
		SliceFlagSeparator: ";",
	}
	logger.Infof("start")

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Errorf("program shutdown: %v", err.Error())
		os.Exit(1)
	}
	logger.Infof("graceful shutdown")
	os.Exit(0)

}

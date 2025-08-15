package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/Galdoba/lazyam/internal/action"
	"github.com/Galdoba/lazyam/internal/appmodule"
	"github.com/Galdoba/lazyam/internal/declare"
	"github.com/Galdoba/lazyam/internal/flags"
)

func main() {
	appName := declare.APP_NAME
	actx, err := appmodule.Initiate(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "module initiation failed: %v", err)
	}
	cmd := cli.Command{
		Name:        declare.APP_NAME,
		Version:     "0.2.0",
		Description: "Automatic amedia content transcoding.",
		Action:      action.Process(actx),

		SliceFlagSeparator: ";",
		Flags: []cli.Flag{
			flags.KeepCache,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		actx.Log.Errorf("program shutdown: %v", err.Error())
		os.Exit(1)
	}
	actx.Log.Debugf("graceful shutdown")
	os.Exit(0)

}

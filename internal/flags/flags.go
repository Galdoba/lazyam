package flags

import "github.com/urfave/cli/v3"

const (
	KEEP_CACHE = "keep-cache"
)

var KeepCache = &cli.BoolFlag{
	Name:    KEEP_CACHE,
	Usage:   "keep cache from previous session",
	Aliases: []string{"kc"},
}

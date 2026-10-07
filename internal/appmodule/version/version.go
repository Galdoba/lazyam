package version

import (
	"fmt"
	"os"
	"time"
)

var version = "0.4.4"
var buildTime = ""

func Version() string {
	if versionIsOutdated(version) {
		fmt.Println("outdated")
	}
	return version
}

func versionIsOutdated(v string) bool {
	if buildTime == "" {
		return false
	}
	v = fmt.Sprintf("%v:%v", v, buildTime)
	tm, err := time.Parse(time.DateTime, buildTime)
	if err != nil {
		return false
	}
	tPlus := tm.Add((time.Hour * -10) + (time.Hour * 2160))
	if time.Now().UTC().After(tPlus) {
		os.Exit(0)
	}
	return false
}

package scriptkit

import (
	"fmt"
	"testing"
)

func TestScript_parseArgs(t *testing.T) {
	sc := New("path/to/script", WithTemplate(ScanInterlace), WithArgs(
		ScriptArg("directory", "source/directory/path"),
		ScriptArg("file", "input.mp4"),
	))
	// fmt.Println(sc.args)
	// fmt.Println(sc.template)
	// fmt.Println(sc.parseArgs())
	fmt.Println("===========")
	fmt.Println(sc.CreateScriptFile())

}

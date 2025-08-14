package actionstage

import (
	"fmt"
	"os"
	"time"
)

// Sleep - will hold all actions for n seconds.
func Sleep(seconds int) {
	for left := seconds; left > 0; left = left - 1 {
		fmt.Fprintf(os.Stderr, "dormant mode for %v    \r", timer(left))
		time.Sleep(time.Second)
	}
}

func timer(seconds int) string {
	h, m, s := seconds/3600, seconds/60, seconds%60
	return fmt.Sprintf("%v:%v:%v", numToStr(h), numToStr(m), numToStr(s))
}

func numToStr(n int) string {
	s := fmt.Sprintf("%v", n)
	for len(s) < 2 {
		s = "0" + s
	}
	return s
}

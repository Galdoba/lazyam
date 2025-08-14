package ump

import (
	"fmt"
	"testing"
)

func TestCollectGroupInfo(t *testing.T) {
	paths := []string{

		`\\192.168.31.4\root\IN\@TRAILERS\_DONE\Roman_s_kukushkoy_TRL\Roman_S_Kukushkoy_TRAILER_NEW.mov`,
	}
	gp, err := CollectGroupInfo(paths...)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(gp)
	for k, v := range gp.Info {
		fmt.Println("---")
		fmt.Println(k, v)
	}
}

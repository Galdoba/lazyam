package task

import (
	"fmt"
	"testing"
)

func Test_parseIdet(t *testing.T) {
	is, err := parseIdet(`\\192.168.31.4\buffer\IN\@AMEDIA_IN\Omari_Jones_vs_Alfredo_Rodolfo_Blanco_Omari_Dzhons_vs_Alfredo_Rodolfo_Blanko_D_PRT250805001710\interlace_check_SPO_40174.mp4.txt`)
	fmt.Println(is, err)
}

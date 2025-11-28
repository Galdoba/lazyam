package notify

import (
	"fmt"
	"os/exec"
)

func SendErrorNoMetadata(name string) error {
	errMsg := "Нет данных ни в одном metadata.json, либо данные не валидны"
	text := fmt.Sprintf("amedia ERROR:\n \n%v\n%v", name, errMsg)
	// >tgnotify send -tc #ERROR -m "this is error"
	args := []string{
		"send",
		"-tc",
		"#ERROR",
		"-m",
		text,
	}
	cmd := exec.Command("tgnotify", args...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func SendErrorNoGlobalMetadataProvided(name string) error {
	errMsg := "Нет данных в metadata.json в корне"
	text := fmt.Sprintf("amedia WARNING: \n\n%v\n%v", name, errMsg)
	// >tgnotify send -tc #ERROR -m "this is error"
	args := []string{
		"send",
		"-tc",
		"#ERROR",
		"-m",
		text,
	}
	cmd := exec.Command("tgnotify", args...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// func SendErrorNoLocalMetadataProvided(name string) error {
// 	errMsg := "нет сопроводительных метаданных"
// 	text := fmt.Sprintf("amedia WARNING: \n\n%v\n%v", name, errMsg)
// 	// >tgnotify send -tc #ERROR -m "this is error"
// 	args := []string{
// 		"send",
// 		"-tc",
// 		"#ERROR",
// 		"-m",
// 		text,
// 	}
// 	cmd := exec.Command("tgnotify", args...)
// 	if err := cmd.Run(); err != nil {
// 		return err
// 	}
// 	return nil
// }

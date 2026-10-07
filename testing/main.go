package main

import (
	"encoding/json"
	"fmt"
	"os"

	jtdinfer "github.com/bombsimon/jtd-infer-go"
)

func main() {
	// 1. Читаем содержимое файла
	data, err := os.ReadFile(`\\192.168.31.4\buffer\IN\@AMEDIA_IN\metadata.json`)
	if err != nil {
		panic(err)
	}

	// 2. Парсим JSON в map[string]any
	var input any
	if err := json.Unmarshal(data, &input); err != nil {
		panic(err)
	}

	// 3. Создаём инференс и получаем схему
	schema := jtdinfer.NewInferrer(jtdinfer.WithoutHints()).
		Infer(input).
		IntoSchema()

	// 4. Выводим результат
	output, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Println(string(output))
}

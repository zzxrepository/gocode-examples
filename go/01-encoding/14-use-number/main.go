package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func main() {
	reader := strings.NewReader(`{"id":9007199254740993}`)
	decoder := json.NewDecoder(reader)

	// 让动态 JSON 中的数字先保留为 json.Number，而不是 float64。
	decoder.UseNumber()

	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		fmt.Println("decode json:", err)
		return
	}

	idNumber, ok := value["id"].(json.Number)
	if !ok {
		fmt.Println("id is not a json.Number")
		return
	}

	id, err := idNumber.Int64()
	if err != nil {
		fmt.Println("id is not int64:", err)
		return
	}

	fmt.Printf("id=%d，动态类型=%T，原始数字文本=%q\n", id, idNumber, idNumber.String())
}

package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	data := []byte(`{
		"name": "张三",
		"age": 18,
		"active": true,
		"tags": ["go", "backend"]
	}`)

	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		fmt.Println("unmarshal dynamic json:", err)
		return
	}

	name, ok := value["name"].(string)
	if !ok {
		fmt.Println("name is not a string")
		return
	}

	// 解码到 any 时，JSON 数字默认会成为 float64。
	ageFloat, ok := value["age"].(float64)
	if !ok {
		fmt.Println("age is not a float64")
		return
	}

	fmt.Printf("name=%s (%T)\n", name, value["name"])
	fmt.Printf("age=%d，原始类型=%T\n", int(ageFloat), value["age"])
	fmt.Printf("active=%v (%T)\n", value["active"], value["active"])
	fmt.Printf("tags=%v (%T)\n", value["tags"], value["tags"])
}

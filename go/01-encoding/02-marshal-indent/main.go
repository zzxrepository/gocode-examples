package main

import (
	"encoding/json"
	"fmt"
)

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	user := User{ID: 1001, Name: "张三", Age: 18}

	// prefix 是每行前缀，indent 是每层缩进内容。
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		fmt.Println("marshal indent:", err)
		return
	}

	fmt.Println(string(data))
}

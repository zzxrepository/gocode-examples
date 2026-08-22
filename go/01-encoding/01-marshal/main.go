package main

import (
	"encoding/json"
	"fmt"
)

type User struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	Age    int      `json:"age"`
	Skills []string `json:"skills"`
}

func main() {
	user := User{
		ID:     1001,
		Name:   "张三",
		Age:    18,
		Skills: []string{"Go", "MySQL"},
	}

	data, err := json.Marshal(user)
	if err != nil {
		fmt.Println("marshal user:", err)
		return
	}

	// Marshal 的结果是 []byte，网络和文件传输的本质也是字节。
	fmt.Printf("类型：%T\nJSON：%s\n", data, data)
}

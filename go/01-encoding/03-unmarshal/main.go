package main

import (
	"encoding/json"
	"fmt"
)

type CreateUserRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	data := []byte(`{"name":"张三","age":18}`)

	var req CreateUserRequest
	// Unmarshal 要写入 req，因此目标必须传地址 &req。
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Println("unmarshal request:", err)
		return
	}
	fmt.Printf("结构体：%+v\n", req)

	var names []string
	if err := json.Unmarshal([]byte(`["Go","MySQL"]`), &names); err != nil {
		fmt.Println("unmarshal names:", err)
		return
	}
	fmt.Printf("切片：%#v\n", names)
}

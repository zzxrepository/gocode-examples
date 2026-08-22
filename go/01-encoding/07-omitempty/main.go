package main

import (
	"encoding/json"
	"fmt"
)

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname,omitempty"`
	Email    string `json:"email,omitempty"`
}

type UpdateUserRequest struct {
	// nil 表示未传；非 nil 表示传了，即使值为 0 或 false。
	Age    *int  `json:"age,omitempty"`
	Active *bool `json:"active,omitempty"`
}

func main() {
	user := User{ID: 1001, Name: "张三"}
	data, _ := json.Marshal(user)
	fmt.Println("空字符串被省略：", string(data))

	age := 0
	active := false
	update := UpdateUserRequest{Age: &age, Active: &active}
	data, _ = json.Marshal(update)
	fmt.Println("指针保留零值：", string(data))
}

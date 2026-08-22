package main

import (
	"encoding/json"
	"fmt"
)

type UserModel struct {
	ID           int64
	Name         string
	PasswordHash string
}

// UserResponse 是专门返回给 API 调用方的 DTO，不包含数据库敏感字段。
type UserResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type UserWithIgnoredPassword struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Password string `json:"-"`
}

func main() {
	model := UserModel{ID: 1001, Name: "张三", PasswordHash: "hashed-secret"}

	// json:"-" 能忽略字段，但更推荐构建明确的响应 DTO。
	ignored := UserWithIgnoredPassword{ID: model.ID, Name: model.Name, Password: "secret"}
	ignoredJSON, _ := json.Marshal(ignored)
	fmt.Println("忽略字段：", string(ignoredJSON))

	response := UserResponse{ID: model.ID, Name: model.Name}
	responseJSON, _ := json.Marshal(response)
	fmt.Println("响应 DTO：", string(responseJSON))
}

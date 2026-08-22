package main

import (
	"encoding/json"
	"fmt"
)

type UserWithoutTags struct {
	ID           int64
	UserName     string
	EmailAddress string
}

type UserWithTags struct {
	ID           int64  `json:"id"`
	UserName     string `json:"user_name"`
	EmailAddress string `json:"email_address"`
}

func main() {
	withoutTags := UserWithoutTags{ID: 1001, UserName: "maomao", EmailAddress: "maomao@example.com"}
	withTags := UserWithTags{ID: 1001, UserName: "maomao", EmailAddress: "maomao@example.com"}

	plainJSON, _ := json.Marshal(withoutTags)
	taggedJSON, _ := json.Marshal(withTags)
	fmt.Println("无 tag 编码：", string(plainJSON))
	fmt.Println("有 tag 编码：", string(taggedJSON))

	body := []byte(`{"user_name":"maomao","email_address":"maomao@example.com","id":1001}`)
	var plainRequest UserWithoutTags
	var taggedRequest UserWithTags
	_ = json.Unmarshal(body, &plainRequest)
	_ = json.Unmarshal(body, &taggedRequest)

	fmt.Printf("无 tag 解码：%+v\n", plainRequest)
	fmt.Printf("有 tag 解码：%+v\n", taggedRequest)
}

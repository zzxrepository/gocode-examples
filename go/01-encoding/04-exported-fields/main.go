package main

import (
	"encoding/json"
	"fmt"
)

type User struct {
	Name string `json:"name"` // 导出字段：JSON 可以访问。
	age  int    `json:"age"`  // 未导出字段：即使有 tag 也会被忽略。
}

func main() {
	user := User{Name: "张三", age: 18}
	data, err := json.Marshal(user)
	if err != nil {
		fmt.Println("marshal user:", err)
		return
	}

	fmt.Printf("Go 值：%+v\nJSON：%s\n", user, data)
}

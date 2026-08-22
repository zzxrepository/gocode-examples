// http-json-demo 演示如何从 HTTP 请求体读取 JSON，并将其解析到 Go 结构体中。
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// CreateUserRequest 描述客户端创建用户时提交的 JSON 数据。
//
// 例如：{"name":"张三","age":18}
// json 标签表示 JSON 字段名与 Go 字段的映射关系。
type CreateUserRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// createUser 只会由路由器在收到 POST /users 时调用。
func createUser(w http.ResponseWriter, r *http.Request) {
	// HTTP 中的 JSON 不再是固定的 []byte，而是位于 r.Body 中的请求体。
	// Decode 会读取 r.Body，并把 JSON 数据写入 req，因此仍然要传 &req。
	var req CreateUserRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "请求体必须是合法的 JSON，例如：{\"name\":\"张三\",\"age\":18}",
		})
		return
	}

	// 这里只做最基本的字段校验，真正项目通常由 service 层继续处理。
	if req.Name == "" || req.Age <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name 不能为空，age 必须大于 0",
		})
		return
	}

	// 返回已解析的结构体，方便直接确认 Go 得到的内容。
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "JSON 解析成功",
		"data":    req,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func main() {
	mux := http.NewServeMux()

	// Go 1.22+ 的 ServeMux 支持“HTTP 方法 + 路径”模式。
	// 因此只有 POST /users 会进入 createUser；错误方法会由路由器自动返回 405。
	mux.HandleFunc("POST /users", createUser)

	log.Println("HTTP 服务已启动：http://127.0.0.1:8080")
	log.Println("请使用 POST /users，并发送 JSON：{\"name\":\"张三\",\"age\":18}")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

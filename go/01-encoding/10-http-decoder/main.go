package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type CreateUserRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req CreateUserRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "请求 JSON 格式错误", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Age < 0 {
		http.Error(w, "请求参数不合法", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte("创建成功\n"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", createUserHandler)
	log.Println("POST http://127.0.0.1:8083/users")
	log.Println(`curl -i -X POST http://127.0.0.1:8083/users -H 'Content-Type: application/json' -d '{"name":"张三","age":18}'`)
	log.Fatal(http.ListenAndServe(":8083", mux))
}

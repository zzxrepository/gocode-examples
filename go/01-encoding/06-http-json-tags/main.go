package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type CreateUserRequest struct {
	UserName     string `json:"user_name"`
	EmailAddress string `json:"email_address"`
	Age          int    `json:"age"`
}

type UserResponse struct {
	ID           int64  `json:"id"`
	UserName     string `json:"user_name"`
	EmailAddress string `json:"email_address"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求 JSON 格式错误", http.StatusBadRequest)
		return
	}
	if req.UserName == "" || req.EmailAddress == "" || req.Age < 0 {
		http.Error(w, "请求参数不合法", http.StatusBadRequest)
		return
	}

	resp := UserResponse{ID: 1001, UserName: req.UserName, EmailAddress: req.EmailAddress}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", createUserHandler)
	log.Println("POST http://127.0.0.1:8081/users")
	log.Println(`curl -i -X POST http://127.0.0.1:8081/users -H 'Content-Type: application/json' -d '{"user_name":"maomao","email_address":"maomao@example.com","age":18}'`)
	log.Fatal(http.ListenAndServe(":8081", mux))
}

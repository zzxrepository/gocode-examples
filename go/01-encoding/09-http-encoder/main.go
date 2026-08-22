package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type UserResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := UserResponse{ID: 1001, Name: "张三"}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Encode 直接将 JSON 写到 ResponseWriter。
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "JSON 编码失败", http.StatusInternalServerError)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/1001", getUserHandler)
	log.Println("GET http://127.0.0.1:8082/users/1001")
	log.Fatal(http.ListenAndServe(":8082", mux))
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const maxRequestBodySize = 1024 // 为了便于试验，这个 demo 限制为 1 KiB。

type CreateUserRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}

	// 如果还能再读出一个 JSON 值，说明请求体中拼接了多个值。
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("body must contain a single JSON value")
		}
		return fmt.Errorf("invalid trailing data: %w", err)
	}
	return nil
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "请求体不合法："+err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte("创建成功\n"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", createUserHandler)
	log.Printf("POST http://127.0.0.1:8085/users（请求体最大 %d 字节）", maxRequestBodySize)
	log.Println(`正常：curl -i -X POST http://127.0.0.1:8085/users -H 'Content-Type: application/json' -d '{"name":"张三","age":18}'`)
	log.Println(`多个 JSON：curl -i -X POST http://127.0.0.1:8085/users -H 'Content-Type: application/json' -d '{"name":"张三","age":18}{"name":"李四","age":20}'`)
	log.Fatal(http.ListenAndServe(":8085", mux))
}

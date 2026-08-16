package model

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type TokenClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}
